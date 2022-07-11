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
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	pb "github.com/infinimesh/proto/handsfree"
)

func makeHFServiceClient(ctx context.Context) (pb.HandsfreeServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewHandsfreeServiceClient(conn), nil
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test different parts of infinimesh",
}

var testHF = &cobra.Command{
	Use:   "hf",
	Short: "Test infinimesh Handsfree service(does connect by default)",
	RunE:  testHFConnect.RunE,
}

var testHFConnect = &cobra.Command{
	Use:   "connect",
	Short: "Test Connect(get codes and wait for data)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeHFServiceClient(ctx)
		if err != nil {
			return err
		}

		c, err := client.Connect(ctx, &pb.ConnectionRequest{AppId: "inf.cli"})
		if err != nil {
			return err
		}

		printJson, _ := cmd.Flags().GetBool("json")
		if !printJson {
			fmt.Println("Streaming started")
		}

		for {
			msg, err := c.Recv()
			if err != nil {
				return err
			}
			if msg.Code == pb.Code_AUTH {
				if len(msg.Payload) < 1 {
					return errors.New("unexpected payload format while reading code")
				}
				if printJson {
					printJsonResponse(map[string]string{
						"code": msg.Payload[0],
					})
				} else {
					fmt.Printf("Enter following code: %s\n", strings.ToUpper(msg.Payload[0]))
				}
			} else if msg.Code == pb.Code_DATA {
				if printJson {
					printJsonResponse(map[string]interface{}{
						"data": msg.Payload,
					})
				} else {
					fmt.Println("Received following payload:")
					for _, el := range msg.Payload {
						fmt.Printf(" - %s\n", el)
					}
				}
				return nil
			} else {
				return fmt.Errorf("received unexpected response, code: %s", msg.Code.String())
			}
		}
	},
}

var testHFSend = &cobra.Command{
	Use:   "send",
	Short: "Test Sending the data",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeHFServiceClient(ctx)
		if err != nil {
			return err
		}

		_, err = client.Send(ctx, &pb.ControlPacket{
			Code:    pb.Code_AUTH,
			Payload: args,
		})

		return err
	},
}

func init() {
	testHF.AddCommand(testHFConnect)
	testHF.AddCommand(testHFSend)

	testCmd.AddCommand(testHF)

	rootCmd.AddCommand(testCmd)
}
