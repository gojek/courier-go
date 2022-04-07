package bootstrap

import (
	"bytes"
	"encoding/base64"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"testing"
)

const fileContent = `
{
  "xds_server": {
    "server_uri": "example.com:443",
    "node": {
      "id": "52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7",
      "cluster": "cluster",
      "metadata": {
        "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": "123456789012345",
        "TRAFFICDIRECTOR_NETWORK_NAME": "thedefault"
      },
      "locality": {
        "zone": "uscentral-5"
      }
    }
  }
}
`

func TestNewConfig(t *testing.T) {
	tests := []struct {
		fileName    string
		fileContent string
		name        string
		want        *Config
		wantErr     bool
	}{
		{
			fileName: "bootstrap-test.json",
			name:     "file_config_success",
			want: &Config{&ServerConfig{
				ServerURI: "example.com:443",
				NodeProto: &envoy_config_core_v3.Node{Id: "52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7",
					Cluster: "cluster",
					Metadata: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": {Kind: &structpb.Value_StringValue{StringValue: "123456789012345"}},
							"TRAFFICDIRECTOR_NETWORK_NAME":       {Kind: &structpb.Value_StringValue{StringValue: "thedefault"}},
						},
					},
					Locality:             &envoy_config_core_v3.Locality{Zone: "uscentral-5"},
					UserAgentName:        courierUserAgentName,
					UserAgentVersionType: &envoy_config_core_v3.Node_UserAgentVersion{UserAgentVersion: grpc.Version},
					ClientFeatures:       []string{"envoy.lb.does_not_support_overprovisioning", "xds.config.resource-in-sotw"},
				},
			}},
		},
		{
			fileContent: fileContent,
			name:        "env_config_success",
			want: &Config{&ServerConfig{
				ServerURI: "example.com:443",
				NodeProto: &envoy_config_core_v3.Node{Id: "52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7",
					Cluster: "cluster",
					Metadata: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": {Kind: &structpb.Value_StringValue{StringValue: "123456789012345"}},
							"TRAFFICDIRECTOR_NETWORK_NAME":       {Kind: &structpb.Value_StringValue{StringValue: "thedefault"}},
						},
					},
					Locality:             &envoy_config_core_v3.Locality{Zone: "uscentral-5"},
					UserAgentName:        courierUserAgentName,
					UserAgentVersionType: &envoy_config_core_v3.Node_UserAgentVersion{UserAgentVersion: grpc.Version},
					ClientFeatures:       []string{"envoy.lb.does_not_support_overprovisioning", "xds.config.resource-in-sotw"},
				},
			}},
		},
		{
			name:    "no_config",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			filecontentBytes := &bytes.Buffer{}
			_, err := base64.NewEncoder(base64.StdEncoding, filecontentBytes).Write([]byte(tt.fileContent))
			if err != nil {
				t.Errorf("Encoding error = %v", err)
				return
			}
			xDSBootstrapFileContent = filecontentBytes.String()
			xDSBootstrapFileName = tt.fileName

			got, err := NewConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareConfigs(tt.want, got) {
				t.Errorf("NewConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareConfigs(a *Config, b *Config) bool {
	if (a == nil) && (b == nil) {
		return true
	}
	return (a.XDSServer.ServerURI == b.XDSServer.ServerURI) && (proto.Equal(a.XDSServer.NodeProto, b.XDSServer.NodeProto))
}
