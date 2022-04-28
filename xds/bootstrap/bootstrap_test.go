package bootstrap

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/gojek/courier-go"
)

const fileName = "bootstrap-test.json"

var fileContent, _ = os.ReadFile(fileName)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		fileName    string
		fileContent string
		name        string
		want        *Config
		wantErr     bool
	}{
		{
			name:     "file_config_success",
			fileName: fileName,
			want: &Config{&ServerConfig{
				ServerURI: "example.com:443",
				NodeProto: &corev3.Node{Id: "52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7",
					Cluster: "cluster",
					Metadata: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": {Kind: &structpb.Value_StringValue{StringValue: "123456789012345"}},
							"TRAFFICDIRECTOR_NETWORK_NAME":       {Kind: &structpb.Value_StringValue{StringValue: "thedefault"}},
						},
					},
					Locality:             &corev3.Locality{Zone: "uscentral-5"},
					UserAgentName:        courierUserAgentName,
					UserAgentVersionType: &corev3.Node_UserAgentVersion{UserAgentVersion: courier.Version()},
					ClientFeatures:       []string{"envoy.lb.does_not_support_overprovisioning", "xds.config.resource-in-sotw"},
				},
			}},
		},
		{
			name:        "env_config_success",
			fileContent: string(fileContent),
			want: &Config{&ServerConfig{
				ServerURI: "example.com:443",
				NodeProto: &corev3.Node{Id: "52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7",
					Cluster: "cluster",
					Metadata: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": {Kind: &structpb.Value_StringValue{StringValue: "123456789012345"}},
							"TRAFFICDIRECTOR_NETWORK_NAME":       {Kind: &structpb.Value_StringValue{StringValue: "thedefault"}},
						},
					},
					Locality:             &corev3.Locality{Zone: "uscentral-5"},
					UserAgentName:        courierUserAgentName,
					UserAgentVersionType: &corev3.Node_UserAgentVersion{UserAgentVersion: courier.Version()},
					ClientFeatures:       []string{"envoy.lb.does_not_support_overprovisioning", "xds.config.resource-in-sotw"},
				},
			}},
		},
		{
			name:        "invalid_config_json",
			fileContent: "randomBytes",
			wantErr:     true,
		},
		{
			name: "invalid_node_config_json",
			fileContent: `
{
  "xds_server": {
    "server_uri": "example.com:443",
    "node": "hello"
  }
}
`,
			wantErr: true,
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

func TestServerConfig_MarshalJSON(t *testing.T) {
	sc := &ServerConfig{
		ServerURI: "localhost:9100",
		NodeProto: &corev3.Node{
			Id:      "52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7",
			Cluster: "cluster",
			Metadata: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER": {Kind: &structpb.Value_StringValue{StringValue: "123456789012345"}},
					"TRAFFICDIRECTOR_NETWORK_NAME":       {Kind: &structpb.Value_StringValue{StringValue: "thedefault"}},
				},
			},
			Locality: &corev3.Locality{
				Region: "asia-east1",
				Zone:   "asia-east1-a",
			},
		},
	}

	b, err := sc.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `{"server_uri":"localhost:9100","server_features":["xds_v3"],"node":{"id":"52fdfc07-2182-454f-963f-5f0f9a621d72~10.9.8.7","cluster":"cluster","metadata":{"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER":"123456789012345","TRAFFICDIRECTOR_NETWORK_NAME":"thedefault"},"locality":{"region":"asia-east1","zone":"asia-east1-a"},"UserAgentVersionType":null}}`, string(b))
}

func TestServerConfig_String(t *testing.T) {
	sc := &ServerConfig{ServerURI: "localhost:9100"}
	assert.Equal(t, "localhost:9100", sc.String())
}
