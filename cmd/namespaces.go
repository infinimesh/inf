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

	pb "github.com/infinimesh/infinimesh/pkg/node/proto"
	accesspb "github.com/infinimesh/infinimesh/pkg/node/proto/access"
	accpb "github.com/infinimesh/infinimesh/pkg/node/proto/accounts"
	nspb "github.com/infinimesh/infinimesh/pkg/node/proto/namespaces"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func makeNamespacesServiceClient(ctx context.Context) (pb.NamespacesServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewNamespacesServiceClient(conn), nil
}

var nsCmd = &cobra.Command{
	Use: "namespaces",
	Short: "Manage infinimesh Namespaces",
	Aliases: []string{"namespace", "ns"},
	RunE: listNsCmd.RunE,
}

var listNsCmd = &cobra.Command{
	Use: "list",
	Short: "List infinimesh Namespaces",
	Aliases: []string{"ls", "l"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeNamespacesServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.List(ctx, &pb.EmptyMessage{})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintNamespacesPool(r.Namespaces)
		return nil
	},
}

var joinsNsCmd = &cobra.Command{
	Use: "joins",
	Short: "List infinimesh Accounts with Access to the given Namespace",
	Aliases: []string{"js", "permissions", "perms"},
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeNamespacesServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.Joins(ctx, &nspb.Namespace{Uuid: args[0]})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintNamespaceJoins(r.Accounts)
		return nil
	},
}

func PrintNamespacesPool(pool []*nspb.Namespace) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"UUID", "Title", "Access"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{ Number: 4, Hidden: true},
	})

	rows := make([]table.Row, len(pool))
	for i, ns := range pool {
		access := ns.Access.Level.String()
		if ns.Access.Role == accesspb.Role_OWNER {
			access += " (owner)"
		}
		rows[i] = table.Row{ns.Uuid, ns.Title, access, int(ns.Access.Level) + int(ns.Access.Role)}
	}
	t.AppendRows(rows)

	t.SortBy([]table.SortBy{
		{Number: 4, Mode: table.DscNumeric},
	})
	t.AppendFooter(table.Row{"", "Total Found", len(pool)}, table.RowConfig{AutoMerge: true})
	t.Render()
}

func PrintNamespaceJoins(pool []*accpb.Account) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"UUID", "Title", "Access"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{ Number: 4, Hidden: true},
	})

	rows := make([]table.Row, len(pool))
	for i, acc := range pool {
		access := acc.Access.Level.String()
		if acc.Access.Role == accesspb.Role_OWNER {
			access += " (owner)"
		}
		rows[i] = table.Row{acc.Uuid, acc.Title, access, int(acc.Access.Level) + int(acc.Access.Role)}
	}
	t.AppendRows(rows)

	t.SortBy([]table.SortBy{
		{Number: 4, Mode: table.DscNumeric},
	})
	t.AppendFooter(table.Row{"", "Total Found", len(pool)}, table.RowConfig{AutoMerge: true})
	t.Render()
}

func init() {
	nsCmd.AddCommand(listNsCmd)
	nsCmd.AddCommand(joinsNsCmd)
	
	rootCmd.AddCommand(nsCmd)
}