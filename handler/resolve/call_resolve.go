package resolveHandler

import (
	"strings"

	"go.scnd.dev/open/nameral/common/config"
	commonGrpc "go.scnd.dev/open/nameral/common/grpc"
	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/module/dns"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (r *Handler) Resolve(server grpc.BidiStreamingServer[proto.ResolveResult, proto.ResolveQuery]) error {
	// Get pre-authenticated client from context
	clientConfig, ok := server.Context().Value(commonGrpc.ClientContextKey).(*config.Client)
	if !ok || clientConfig == nil {
		return status.Error(codes.Unauthenticated, "client not authenticated")
	}

	// Get zones from metadata
	md, ok := metadata.FromIncomingContext(server.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, "missing metadata")
	}
	zonesRaw := md["zones"]
	if len(zonesRaw) == 0 {
		return status.Error(codes.InvalidArgument, "missing zones metadata")
	}
	requestedZones := strings.Split(zonesRaw[0], ",")

	// Intersect with allowed zones; "." means allow all zones
	allowedSet := make(map[string]bool)
	allowAll := false
	for _, z := range clientConfig.AllowedZones {
		if *z == "." {
			allowAll = true
		} else {
			allowedSet[*z] = true
		}
	}
	var finalZones []string
	for _, z := range requestedZones {
		z = strings.TrimSpace(z)
		if allowAll || allowedSet[z] {
			finalZones = append(finalZones, z)
		}
	}
	if len(finalZones) == 0 {
		return status.Error(codes.PermissionDenied, "no allowed zones")
	}

	// Build client stream and register
	cs := &dns.ClientStream{
		Name:   *clientConfig.Name,
		Dns:    r.Dns,
		Stream: server,
	}
	r.Dns.Register(cs, finalZones)
	defer r.Dns.Unregister(cs, finalZones)

	// Read loop: dispatch incoming results to pending waiters
	for {
		result, err := server.Recv()
		if err != nil {
			return nil
		}
		cs.Deliver(result.No, result)
	}
}
