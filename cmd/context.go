/*
Copyright © 2021-2022 Nikita Ivanovski info@slnt-opp.xyz

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"

	pb "github.com/infinimesh/proto/node"
	accpb "github.com/infinimesh/proto/node/accounts"
	"github.com/infinimesh/proto/node/sessions"
)

func getVersion() string {
	return VERSION
}

type ContextConf struct {
	Selected string `yaml:"selected"`
}

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "Print current infinimesh CLI Context | Aliases: ctx",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := make(map[string]interface{})
		data["version"] = getVersion()

		data["host"] = viper.Get("infinimesh")
		if data["host"] == nil {
			data = map[string]interface{}{
				"error": "No infinimesh context found",
			}
		}

		if insec := viper.GetBool("insecure"); insec {
			data["insecure"] = insec
		}

		if viper.GetString("context") != "" {
			data["context"] = viper.GetString("context")
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(data)
		}

		caser := cases.Title(language.English)
		for k, v := range data {
			fmt.Printf("%s: %v\n", caser.String(k), v)
		}

		return nil
	},
}

var setContextCmd = &cobra.Command{
	Use:   "set-context <context>",
	Short: "Set infinimesh CLI Context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		profilesCfg := fmt.Sprintf("%s/.infinimesh.contexts", home)

		if _, err := os.Stat(profilesCfg); os.IsNotExist(err) {
			if _, err := os.Create(profilesCfg); err != nil { // perm 0666
				fmt.Fprintln(os.Stderr, "Can't create default contexts config file")
				panic(err)
			}
		}

		contexts := ContextConf{
			Selected: args[0],
		}
		r, _ := yaml.Marshal(contexts)
		return os.WriteFile(profilesCfg, r, 0640)
	},
}

var loginCmd = &cobra.Command{
	Use:     "login <host:port> <login>",
	Aliases: []string{"l", "auth", "a"},
	Args:    cobra.ExactArgs(2),
	Short:   "Authorize in infinimesh and store credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		insec, _ := cmd.Flags().GetBool("insecure")
		if insec {
			creds = insecure.NewCredentials()
		}
		conn, err := grpc.Dial(args[0], grpc.WithTransportCredentials(creds))
		if err != nil {
			return err
		}

		client := pb.NewAccountsServiceClient(conn)

		t := "standard"
		if ok, err := cmd.Flags().GetBool("ldap"); err != nil {
			return err
		} else if ok {
			t = "ldap"
		}

		var password string
		if p, _ := cmd.Flags().GetString("password"); p != "" {
			password = p
		} else {

			prompt := promptui.Prompt{
				Label: "Password",
				Mask:  '*',
			}

			password, err = prompt.Run()
			if err != nil {
				return err
			}
		}

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		client_id := fmt.Sprintf("CLI | %s | %s", hostname, runtime.GOOS)
		req := &pb.TokenRequest{
			Auth: &accpb.Credentials{
				Type: t, Data: []string{args[1], password},
			},
			Client: &client_id,
		}

		res, err := client.Token(context.Background(), req)
		if err != nil {
			return err
		}
		token := res.GetToken()
		printToken, _ := cmd.Flags().GetBool("print-token")
		if printToken {
			fmt.Println(token)
		}

		infContext := "default"
		if ctx, _ := cmd.Flags().GetString("context"); ctx != "" {
			infContext = ctx
		}

		viper.Set("infinimesh", args[0])
		viper.Set("token", token)
		viper.Set("insecure", insec)
		viper.Set("context", infContext)

		err = viper.WriteConfig()
		return err
	},
}

// make SessionsServiceClient
func makeSessionsServiceClient(ctx context.Context) (pb.SessionsServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewSessionsServiceClient(conn), nil
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage Sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeSessionsServiceClient(ctx)
		if err != nil {
			return err
		}

		sess, err := client.Get(ctx, &pb.EmptyMessage{})
		if err != nil {
			return err
		}

		var act *sessions.Activity
		if wa, _ := cmd.Flags().GetBool("with-activity"); wa {
			act, err = client.GetActivity(ctx, &pb.EmptyMessage{})
			if err != nil {
				return err
			}
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(map[string]interface{}{
				"sessions": sess.GetSessions(),
				"activity": act.GetLastSeen(),
			})
		}

		PrintSessions(sess.GetSessions(), act)
		return nil
	},
}

var sessionRevokeCmd = &cobra.Command{
	Use: "revoke [...sid]",
	Aliases: []string{
		"delete",
		"remove",
	},
	Short: "Revoke the Session(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeSessionsServiceClient(ctx)
		if err != nil {
			return err
		}

		for _, sid := range args {
			_, err = client.Revoke(ctx, &sessions.Session{Id: sid})
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func PrintSessions(pool []*sessions.Session, act *sessions.Activity) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	header := []interface{}{"ID", "Client", "Created", "Expires"}
	if act != nil {
		header = append(header, "Last Seen")
	}
	t.AppendHeader(header)

	rows := make([]table.Row, len(pool))
	for i, sess := range pool {
		client := sess.GetClient()
		if client == "" {
			client = "N/A"
		}

		expires := sess.GetExpires().AsTime().Format(time.RFC1123)
		if sess.GetExpires() == nil {
			expires = "Never"
		}

		rows[i] = table.Row{sess.GetId(), client, sess.GetCreated().AsTime().Format(time.RFC1123), expires}

		if act != nil {
			last_seen := "Never"
			if ts, ok := act.GetLastSeen()[sess.GetId()]; ok {
				last_seen = ts.AsTime().Format(time.RFC1123)
			}

			rows[i] = append(rows[i], last_seen)
		}

		if sess.GetCurrent() {
			rows[i] = append(rows[i], "Current")
		}
	}

	t.AppendRows(rows)

	t.SortBy([]table.SortBy{
		{Name: "Expires", Mode: table.Asc},
		{Name: "Created", Mode: table.Dsc},
	})

	t.AppendFooter(table.Row{"", "", "Total Found", len(pool)})
	t.Render()
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print infinimesh CLI version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			data, err := json.Marshal(map[string]string{
				"version": getVersion(),
			})
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("CLI Version:", getVersion())
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("password", "p", "", "Password for Standard Credentials")
	loginCmd.Flags().Bool("print-token", false, "")
	loginCmd.Flags().Bool("insecure", false, "Use WithInsecure instead of TLS")
	loginCmd.Flags().Bool("ldap", false, "Use Credentials Type LDAP")

	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(setContextCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(versionCmd)

	sessionsCmd.AddCommand(sessionRevokeCmd)

	sessionsCmd.Flags().BoolP("with-activity", "a", false, "Show Activity")
	rootCmd.AddCommand(sessionsCmd)
}
