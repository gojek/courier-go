package bootstrap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/protobuf/proto"

	"github.com/gojekfarm/courier-go"
)

const (
	// The "server_features" field in the bootstrap file contains a list of
	// features supported by the server. A value of "xds_v3" indicates that the
	// server supports the v3 version of the xDS transport protocol.
	serverFeaturesV3 = "xds_v3"

	courierUserAgentName            = "Courier Go"
	clientFeatureNoOverprovisioning = "envoy.lb.does_not_support_overprovisioning"
	clientFeatureResourceWrapper    = "xds.config.resource-in-sotw"

	xDSBootstrapFileNameEnvVar    = "COURIER_XDS_BOOTSTRAP"
	xDSBootstrapFileContentEnvVar = "COURIER_XDS_BOOTSTRAP_CONFIG_BASE64"
)

var (
	xDSBootstrapFileName    = os.Getenv(xDSBootstrapFileNameEnvVar)
	xDSBootstrapFileContent = os.Getenv(xDSBootstrapFileContentEnvVar)

	jsonUnmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
)

func readXDSBootstrapConfig() ([]byte, error) {
	if fn := strings.TrimSpace(xDSBootstrapFileName); len(fn) != 0 {
		return os.ReadFile(fn)
	}

	if fc := strings.TrimSpace(xDSBootstrapFileContent); len(fc) != 0 {
		return base64.StdEncoding.DecodeString(fc)
	}

	return nil, fmt.Errorf("none of the bootstrap environment variables (%q or %q) defined",
		xDSBootstrapFileNameEnvVar, xDSBootstrapFileContentEnvVar)
}

// Config provides the xDS client with several key bits of information that it
// requires in its interaction with the management server. The Config is
// initialized from the bootstrap file.
type Config struct {
	// XDSServer is the management server to connect to.
	XDSServer *ServerConfig `json:"xds_server"`
}

// NewConfig returns a bootstrap config after reading the BootstrapConfig from the
// file specified by COURIER_XDS_BOOTSTRAP env var or the base64 encoded string in
// COURIER_XDS_BOOTSTRAP_CONFIG_BASE64 var.
func NewConfig() (*Config, error) {
	data, err := readXDSBootstrapConfig()
	if err != nil {
		return nil, err
	}

	return NewConfigFromContents(data)
}

// NewConfigFromContents returns a bootstrap config after reading the BootstrapConfig from data.
func NewConfigFromContents(data []byte) (*Config, error) {
	cfg := new(Config)

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("xds: Failed to parse bootstrap config: %v", err)
	}

	return cfg, nil
}

type xdsNode struct {
	node proto.Message
}

type xdsServer struct {
	ServerURI      string   `json:"server_uri"`
	ServerFeatures []string `json:"server_features"`
	Node           xdsNode  `json:"node"`
}

// ServerConfig contains the configuration to connect to a server, including
// URI, creds, and transport API version (e.g. v2 or v3).
type ServerConfig struct {
	// ServerURI is the management server to connect to.
	//
	// The bootstrap file contains an ordered list of xDS servers to contact for
	// this authority. The first one is picked.
	ServerURI string `json:"server_uri"`
	// NodeProto contains the Node proto to be used in xDS requests. The actual
	// type depends on the transport protocol version used.
	//
	// Note that it's specified in the bootstrap globally for all the servers,
	// but we keep it in each server config so that its type (e.g. *v3pb.Node)
	// is consistent with the transport API version.
	NodeProto proto.Message
}

// String returns the string representation of the ServerConfig.
//
// This string representation will be used as map keys in federation
// (`map[ServerConfig]authority`), so that the xDS ClientConn and stream will be
// shared by authorities with different names but the same server config.
//
// It covers (almost) all the fields so the string can represent the config
// content. It doesn't cover NodeProto because NodeProto isn't used by
// federation.
func (sc *ServerConfig) String() string {
	return strings.Join([]string{sc.ServerURI}, "-")
}

// MarshalJSON marshals the ServerConfig to json.
func (sc ServerConfig) MarshalJSON() ([]byte, error) {
	server := xdsServer{
		ServerURI: sc.ServerURI,
		Node:      xdsNode{node: sc.NodeProto},
	}
	server.ServerFeatures = []string{serverFeaturesV3}

	return json.Marshal(server)
}

// UnmarshalJSON takes the json data (a server) and unmarshals it to the struct.
func (sc *ServerConfig) UnmarshalJSON(data []byte) error {
	var server xdsServer
	if err := json.Unmarshal(data, &server); err != nil {
		return fmt.Errorf("xds: json.Unmarshal(data) for field ServerConfig failed during bootstrap: %v", err)
	}

	sc.ServerURI = server.ServerURI
	sc.NodeProto = server.Node.node

	return nil
}

// MarshalJSON marshals the ServerConfig to json.
func (n xdsNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.node)
}

// UnmarshalJSON takes the json data (a server) and unmarshals it to the struct.
func (n *xdsNode) UnmarshalJSON(data []byte) error {
	in := new(corev3.Node)
	if err := jsonUnmarshaler.Unmarshal(bytes.NewReader(data), in); err != nil {
		return fmt.Errorf("xds: json.Unmarshal(data) for field NodeProto failed during bootstrap: %v", err)
	}

	in.UserAgentName = courierUserAgentName
	in.UserAgentVersionType = &corev3.Node_UserAgentVersion{UserAgentVersion: courier.Version()}
	in.ClientFeatures = append(in.ClientFeatures, clientFeatureNoOverprovisioning, clientFeatureResourceWrapper)

	n.node = in

	return nil
}
