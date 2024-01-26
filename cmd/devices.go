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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/infinimesh/proto/node/access"
	"os"
	"strings"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/infinimesh/infinimesh/pkg/convert"
	pb "github.com/infinimesh/proto/node"
	devpb "github.com/infinimesh/proto/node/devices"
	shadowpb "github.com/infinimesh/proto/shadow"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func makeDevicesServiceClient(ctx context.Context) (pb.DevicesServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewDevicesServiceClient(conn), nil
}

func makeShadowServiceClient(ctx context.Context) (pb.ShadowServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewShadowServiceClient(conn), nil
}

// devicesCmd represents the devices command
var devicesCmd = &cobra.Command{
	Use:     "devices",
	Short:   "Manage infinimesh Devices",
	Aliases: []string{"device", "dev"},
	RunE:    listDevicesCmd.RunE,
}

var listDevicesCmd = &cobra.Command{
	Use:     "list",
	Short:   "List infinimesh devices",
	Aliases: []string{"ls", "l"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeDevicesServiceClient(ctx)
		if err != nil {
			return err
		}

		req := &pb.QueryRequest{}
		if ns, err := cmd.Flags().GetString("ns"); err != nil && ns != "" {
			req.Namespace = &ns
		}

		r, err := client.List(ctx, req)
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintDevicesPool(r.Devices)
		return nil
	},
}

var getDeviceCmd = &cobra.Command{
	Use:   "get",
	Short: "Get infinimesh device",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeDevicesServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.Get(ctx, &devpb.Device{Uuid: args[0]})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintSingleDevice(r)
		return nil
	},
}

var toggleDeviceCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle infinimesh device (enabled/disabled)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeDevicesServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.Get(ctx, &devpb.Device{Uuid: args[0]})
		if err != nil {
			return err
		}

		r.Enabled = !r.Enabled
		r, err = client.Toggle(ctx, r)
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		res := "disabled"
		if r.Enabled {
			res = "enabled"
		}
		fmt.Println("Device is now: " + res)

		return nil
	},
}

var makeDeviceTokenCmd = &cobra.Command{
	Use:     "token",
	Short:   "Make device token",
	Aliases: []string{"tok", "t"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeDevicesServiceClient(ctx)
		if err != nil {
			return err
		}

		var devices = make(map[string]access.Level, len(args))

		for _, arg := range args {
			devices[arg] = access.Level_NONE
		}

		r, err := client.MakeDevicesToken(ctx, &pb.DevicesTokenRequest{
			Devices: devices,
		})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		fmt.Println(r.Token)
		return nil
	},
}

var createDeviceCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create infinimesh device",
	Aliases: []string{"add", "a", "new", "crt"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeDevicesServiceClient(ctx)
		if err != nil {
			return err
		}

		if _, err := os.Stat(args[0]); os.IsNotExist(err) {
			return errors.New("Template doesn't exist at path " + args[0])
		}

		var format string
		{
			pathSlice := strings.Split(args[0], ".")
			format = pathSlice[len(pathSlice)-1]
		}

		template, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Println("Error while reading template")
			return err
		}

		switch format {
		case "json":
		case "yml", "yaml":
			template, err = convert.ConvertBytes(template)
		default:
			return errors.New("Unsupported template format " + format)
		}

		if err != nil {
			fmt.Println("Error while parsing template")
			return err
		}

		fmt.Println("Template", string(template))

		var device devpb.Device
		err = json.Unmarshal(template, &device)
		if err != nil {
			fmt.Println("Error while parsing template")
			return err
		}

		soft, _ := cmd.Flags().GetBool("soft")
		if !soft {
			certPath, _ := cmd.Flags().GetString("crt")
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				return errors.New("Certificate doesn't exist at path " + certPath)
			}
			pem, err := os.ReadFile(certPath)
			if err != nil {
				fmt.Println("Error while reading certificate")
				return err
			}

			cert := &devpb.Certificate{
				PemData: string(pem),
			}
			device.Certificate = cert
		}

		ns, _ := cmd.Flags().GetString("namespace")

		res, err := client.Create(ctx, &devpb.CreateRequest{
			Device:    &device,
			Namespace: ns,
		})
		if err != nil {
			return err
		}

		fmt.Println("Device Created, UUID:", res.Device.Uuid)
		return nil
	},
}

var mgmtDeviceStateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage device state",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()

		var token string
		if t, _ := cmd.Flags().GetString("token"); t != "" {
			token = t
		} else {
			client, err := makeDevicesServiceClient(ctx)
			if err != nil {
				return err
			}

			var devices = make(map[string]access.Level, len(args))

			for _, arg := range args {
				devices[arg] = access.Level_NONE
			}

			r, err := client.MakeDevicesToken(ctx, &pb.DevicesTokenRequest{
				Devices: devices,
			})

			if err != nil {
				return err
			}
			token = r.Token
		}

		ctx = metadata.AppendToOutgoingContext(context.Background(), "Authorization", "Bearer "+token)
		client, err := makeShadowServiceClient(ctx)
		if err != nil {
			return err
		}

		if patch, _ := cmd.Flags().GetString("patch"); patch != "" {
			req := &shadowpb.Shadow{
				Device: args[0],
				Desired: &shadowpb.State{
					Data: &structpb.Struct{},
				},
			}

			err = req.Desired.Data.UnmarshalJSON([]byte(patch))
			if err != nil {
				return err
			}

			_, err = client.Patch(ctx, req)
			if err != nil {
				return err
			}
		}

		if report, _ := cmd.Flags().GetString("report"); report != "" {
			req := &shadowpb.Shadow{
				Device: args[0],
				Reported: &shadowpb.State{
					Data: &structpb.Struct{},
				},
			}

			err = req.Reported.Data.UnmarshalJSON([]byte(report))
			if err != nil {
				fmt.Printf("Attempted Report: '%s'\n", report)
				return err
			}

			_, err = client.Patch(ctx, req)
			if err != nil {
				return err
			}
		}

		if remove, _ := cmd.Flags().GetString("remove"); remove != "" {
			input := strings.SplitN(remove, ".", 2)
			req := &shadowpb.RemoveRequest{
				Device: args[0],
			}
			if input[0] == "reported" {
				req.StateKey = shadowpb.StateKey_REPORTED
			} else {
				req.StateKey = shadowpb.StateKey_DESIRED
			}

			req.Key = input[1]

			state, err := client.Remove(ctx, req)
			if err != nil {
				return err
			}

			PrintSingleDeviceState(state)
			return nil
		}

		if stream, _ := cmd.Flags().GetBool("stream"); stream {
			delta, _ := cmd.Flags().GetBool("delta")
			sync, _ := cmd.Flags().GetBool("sync")
			c, err := client.StreamShadow(ctx, &shadowpb.StreamShadowRequest{OnlyDelta: delta, Sync: sync})
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
				if printJson {
					printJsonResponse(msg)
				} else {
					PrintSingleDeviceState(msg)
				}
			}
		}

		r, err := client.Get(ctx, &shadowpb.GetRequest{
			Pool: args,
		})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		for _, shadow := range r.GetShadows() {
			PrintSingleDeviceState(shadow)
		}
		return nil
	},
}

var mgmtDevIceStateMQTTCmd = &cobra.Command{
	Use:   "mqtt",
	Short: "Manage device state via MQTT",
	RunE: func(cmd *cobra.Command, args []string) error {

		opts := MQTT.NewClientOptions()

		broker, _ := cmd.Flags().GetString("host")
		if broker == "" {
			broker = strings.Replace(
				strings.Split(viper.GetString("infinimesh"), ":")[0],
				"api.", "mqtt.", 1)
		}

		port, _ := cmd.Flags().GetString("port")
		if basic, _ := cmd.Flags().GetString("basic"); basic != "" {
			cred := strings.Split(basic, ":")
			opts.SetUsername(cred[0])
			opts.SetPassword(cred[1])

			if port == "" {
				port = "1883"
			}
			broker = "mqtt://" + broker + ":" + port
		} else {
			cert_path, _ := cmd.Flags().GetString("crt")
			if cert_path == "" {
				return errors.New("no certificate given")
			}
			key_path, _ := cmd.Flags().GetString("key")
			if key_path == "" {
				return errors.New("no key given")
			}

			cert, err := tls.LoadX509KeyPair(cert_path, key_path)
			if err != nil {
				return err
			}

			opts.SetTLSConfig(&tls.Config{
				Certificates:       []tls.Certificate{cert},
				ClientAuth:         tls.NoClientCert,
				ClientCAs:          nil,
				InsecureSkipVerify: true,
			})

			if port == "" {
				port = "8883"
			}
			broker = "mqtts://" + broker + ":" + port
		}

		opts.AddBroker(broker)

		client_id, _ := cmd.Flags().GetString("client-id")
		opts.SetClientID(client_id)

		client := MQTT.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}

		desired, _ := cmd.Flags().GetBool("desired")
		if desired {
			messages := make(chan string, 2)
			token := client.Subscribe("devices/+/state/desired/delta", 1, func(_ MQTT.Client, msg MQTT.Message) {
				messages <- string(msg.Payload())
			})

			if token.Wait() && token.Error() != nil {
				return token.Error()
			}

			for msg := range messages {
				fmt.Println(msg)
			}
		}

		report, _ := cmd.Flags().GetString("report")
		if report != "" {
			client.Publish("devices/+/state/reported/delta", 1, false, report)
			client.Disconnect(250)
		}

		return nil
	},
}

func PrintSingleDevice(d *devpb.Device) {
	fmt.Printf("UUID: %s\n", d.Uuid)
	fmt.Printf("Title: %s\n", d.Title)
	fmt.Printf("Enabled: %t\n", d.Enabled)

	tags := strings.Join(d.Tags, ",")
	if tags == "" {
		tags = "-"
	}
	fmt.Printf("Tags: %s\n", tags)

	fingerprint := hex.EncodeToString(d.Certificate.Fingerprint)
	fmt.Printf("Fingerprint:\n  Algorythm: %s\n  Hash: %s\n", d.Certificate.Algorithm, fingerprint)
}

func PrintSingleDeviceState(state *shadowpb.Shadow) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendRow(table.Row{"Device", state.Device})
	t.AppendHeader(table.Row{"State", "Reported", "Desired"})

	var reported []byte
	var reported_time string
	if state.Reported == nil {
		reported = []byte("{}")
		reported_time = "-"
	} else {
		reported, _ = state.Reported.Data.MarshalJSON()
		reported_time = state.Reported.Timestamp.AsTime().String()
	}

	var desired []byte
	var desired_time string
	if state.Desired == nil {
		desired = []byte("{}")
		desired_time = "-"
	} else {
		desired, _ = state.Desired.Data.MarshalJSON()
		desired_time = state.Desired.Timestamp.AsTime().String()
	}
	t.AppendRow(table.Row{"Data", string(reported), string(desired)})
	t.AppendRow(table.Row{"Timestamp", reported_time, desired_time})

	t.Render()
}

func PrintDevicesPool(pool []*devpb.Device) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"UUID", "Title", "Enabled", "Tags"})

	rows := make([]table.Row, len(pool))
	for i, dev := range pool {
		tags := strings.Join(dev.Tags, ",")
		if tags == "" {
			tags = "-"
		}
		rows[i] = table.Row{dev.Uuid, dev.Title, dev.Enabled, tags}
	}
	t.AppendRows(rows)

	t.SortBy([]table.SortBy{
		{Name: "UUID", Mode: table.Asc},
	})

	t.AppendFooter(table.Row{"", "", "Total Found", len(pool)}, table.RowConfig{AutoMerge: true})
	t.Render()
}

func init() {

	listDevicesCmd.Flags().String("ns", "", "Namespace to list devices from")
	devicesCmd.AddCommand(listDevicesCmd)

	devicesCmd.AddCommand(getDeviceCmd)

	makeDeviceTokenCmd.Flags().Bool("allow-post", false, "Allow posting devices states")
	devicesCmd.AddCommand(makeDeviceTokenCmd)

	createDeviceCmd.Flags().String("crt", "", "Path to certificate file")
	createDeviceCmd.Flags().StringP("namespace", "n", "", "Namespace to create device in")
	devicesCmd.AddCommand(createDeviceCmd)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("infinimesh-cli-%d", os.Getpid())
	}

	mgmtDevIceStateMQTTCmd.Flags().StringP("crt", "c", "", "Path to certificate file")
	mgmtDevIceStateMQTTCmd.Flags().StringP("key", "k", "", "Path to private key file")
	mgmtDevIceStateMQTTCmd.Flags().String("host", "", "MQTT broker Host and Port")
	mgmtDevIceStateMQTTCmd.Flags().StringP("basic", "b", "", "MQTT Basic Auth string (login:pass)")
	mgmtDevIceStateMQTTCmd.Flags().StringP("client-id", "i", hostname, "MQTT client id")
	mgmtDevIceStateMQTTCmd.Flags().StringP("report", "r", "", "Report Device state")
	mgmtDevIceStateMQTTCmd.Flags().BoolP("desired", "d", false, "Subscribe to Device Desired state")
	mgmtDeviceStateCmd.AddCommand(mgmtDevIceStateMQTTCmd)

	mgmtDeviceStateCmd.Flags().BoolP("delta", "d", false, "Wether to stream only delta")
	mgmtDeviceStateCmd.Flags().Bool("sync", false, "Wether to send current state upon connection")
	mgmtDeviceStateCmd.Flags().BoolP("stream", "s", false, "Stream device state")
	mgmtDeviceStateCmd.Flags().StringP("patch", "p", "", "Patch Device Desired state")
	mgmtDeviceStateCmd.Flags().StringP("report", "r", "", "Report Device state")
	mgmtDeviceStateCmd.Flags().String("remove", "", "Remove Device state key as <reported|desired>.<key>")
	mgmtDeviceStateCmd.Flags().StringP("token", "t", "", "Device token(new would be obtained if not present)")
	devicesCmd.AddCommand(mgmtDeviceStateCmd)

	devicesCmd.AddCommand(toggleDeviceCmd)

	rootCmd.AddCommand(devicesCmd)
}
