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
	"os"

	pb "github.com/infinimesh/proto/node"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func makeInternalServiceClient(ctx context.Context) (pb.InternalServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewInternalServiceClient(conn), nil
}

var interalServiceCmd = &cobra.Command{
	Use:     "internal",
	Aliases: []string{"int", "i"},
	Short:   "Platform internals helper commands",
}

var ldapProvidersCmd = &cobra.Command{
	Use:   "ldap",
	Short: "Get LDAP Providers registered",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeInternalServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.GetLDAPProviders(ctx, &pb.EmptyMessage{})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Key", "Title"})

		for key, title := range r.Providers {
			t.AppendRow(table.Row{key, title})
		}

		t.Render()

		return nil
	},
}

func init() {

	interalServiceCmd.AddCommand(ldapProvidersCmd)

	rootCmd.AddCommand(interalServiceCmd)
}
