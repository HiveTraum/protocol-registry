package interceptors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/user/protocol_registry/internal/entities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mutatingMethods lists the RPC full method names that require authentication.
var mutatingMethods = map[string]bool{
	"/registry.v1.ProtocolRegistry/PublishProtocol":    true,
	"/registry.v1.ProtocolRegistry/RegisterConsumer":   true,
	"/registry.v1.ProtocolRegistry/UnregisterConsumer": true,
}

// AuthInterceptor validates API keys on mutating RPCs when auth is enabled.
type AuthInterceptor struct {
	repo    entities.APIKeyRepository
	enabled bool
}

func NewAuthInterceptor(repo entities.APIKeyRepository, enabled bool) *AuthInterceptor {
	return &AuthInterceptor{repo: repo, enabled: enabled}
}

func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !a.enabled || !mutatingMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		vals := md.Get("x-api-key")
		if len(vals) == 0 || vals[0] == "" {
			return nil, status.Error(codes.Unauthenticated, "missing api key")
		}

		rawKey := vals[0]
		hash := sha256Key(rawKey)

		apiKey, err := a.repo.FindByHash(ctx, hash)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "auth lookup failed: %v", err)
		}
		if apiKey == nil || apiKey.RevokedAt != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or revoked api key")
		}

		return handler(ctx, req)
	}
}

func sha256Key(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
