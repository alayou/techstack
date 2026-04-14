package depsdev

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	pb "deps.dev/api/v3"
	pdv3 "deps.dev/api/v3alpha"
	"deps.dev/util/resolve"
	"github.com/google/osv-scalibr/clients/datasource"
	"github.com/google/osv-scalibr/depsdev"
	"github.com/google/osv-scalibr/purl"
	scalibrversion "github.com/google/osv-scalibr/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// DepsDevClient is a ResolutionClient wrapping the official resolve.APIClient
type DepsDevClient struct {
	resolve.APIClient
	c  *datasource.CachedInsightsClient
	v3 pdv3.InsightsClient
}

// NewDepsDevClient creates a new DepsDevClient.
func NewClient() (*DepsDevClient, error) {
	userAgent := "osv-scalibr/" + scalibrversion.ScannerVersion
	c, err := datasource.NewCachedInsightsClient(depsdev.DepsdevAPI, userAgent)
	if err != nil {
		return nil, err
	}
	connv3, err := newV3Client(depsdev.DepsdevAPI, userAgent)
	if err != nil {
		return nil, err
	}
	v3 := pdv3.NewInsightsClient(connv3)
	return &DepsDevClient{APIClient: *resolve.NewAPIClient(c), c: c, v3: v3}, nil
}

func newV3Client(addr, userAgent string) (*grpc.ClientConn, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("getting system cert pool: %w", err)
	}
	creds := credentials.NewClientTLSFromCert(certPool, "")
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	if userAgent != "" {
		dialOpts = append(dialOpts, grpc.WithUserAgent(userAgent))
	}
	return grpc.NewClient(addr, dialOpts...)
}

func (s *DepsDevClient) GetProjectPackageVersions(ctx context.Context, id string) (*pb.ProjectPackageVersions, error) {
	return s.c.GetProjectPackageVersions(ctx, &pb.GetProjectPackageVersionsRequest{
		ProjectKey: &pb.ProjectKey{
			Id: id,
		},
	})
}

func (s *DepsDevClient) GetProject(ctx context.Context, id string) (*pb.Project, error) {
	return s.c.GetProject(ctx, &pb.GetProjectRequest{
		ProjectKey: &pb.ProjectKey{
			Id: id,
		},
	})
}
func (s *DepsDevClient) GetPackage(ctx context.Context, system, name string) (*pb.Package, error) {
	return s.c.GetPackage(ctx, &pb.GetPackageRequest{
		PackageKey: &pb.PackageKey{
			System: ToSystem(system),
			Name:   name,
		},
	})
}

func (s *DepsDevClient) Version(ctx context.Context, system, name, version string) (*pb.Version, error) {
	if strings.ToLower("system") == "golang" {
		system = "go"
	}
	if !strings.HasPrefix(version, "v") && strings.ToLower("system") == "go" {
		version = "v" + version
	}
	return s.c.GetVersion(ctx, &pb.GetVersionRequest{
		VersionKey: &pb.VersionKey{
			System:  ToSystem(system),
			Name:    name,
			Version: version,
		},
	})
}

func (s *DepsDevClient) QueryByName(ctx context.Context, system, name, version string) (*pb.QueryResult, error) {
	if strings.ToLower("system") == "golang" {
		system = "go"
	}
	if !strings.HasPrefix(version, "v") && strings.ToLower("system") == "go" {
		version = "v" + version
	}
	return s.c.Query(ctx, &pb.QueryRequest{
		VersionKey: &pb.VersionKey{
			System:  ToSystem(system),
			Name:    name,
			Version: version,
		},
	})
}

func (s *DepsDevClient) QueryByHash(ctx context.Context, ty, value string) (*pb.QueryResult, error) {
	return s.c.Query(ctx, &pb.QueryRequest{
		Hash: &pb.Hash{
			Type:  pb.HashType(pb.HashType_value[strings.ToUpper(ty)]),
			Value: []byte(value),
		},
	})
}

type VersionBatchReq struct {
	System  string
	Name    string
	Version string
}

func (s *DepsDevClient) GetVersionBatch(ctx context.Context, pageToken string, in []VersionBatchReq) (out []*pdv3.VersionBatch_Response, err error) {
	var reqs = make([]*pdv3.GetVersionRequest, len(in))
	for i := range reqs {
		reqs[i] = &pdv3.GetVersionRequest{
			VersionKey: &pdv3.VersionKey{
				Name:    in[i].Name,
				System:  pdv3.System(ToSystem(in[i].System)),
				Version: in[i].Version,
			},
		}
	}
	var res *pdv3.VersionBatch
	res, err = s.v3.GetVersionBatch(ctx, &pdv3.GetVersionBatchRequest{
		Requests:  reqs,
		PageToken: pageToken,
	})
	if err != nil {
		return
	}
	out = res.Responses
	if pageToken != "" {
		var ls []*pdv3.VersionBatch_Response
		ls, err = s.GetVersionBatch(ctx, pageToken, in)
		if err != nil {
			return
		}
		out = append(out, ls...)
	}
	return
}

func (s *DepsDevClient) GetProjectsBatch(ctx context.Context, pageToken string, in []string) (out []*pdv3.ProjectBatch_Response, err error) {
	var reqs = make([]*pdv3.GetProjectRequest, len(in))
	for i := range reqs {
		reqs[i] = &pdv3.GetProjectRequest{
			ProjectKey: &pdv3.ProjectKey{
				Id: in[i],
			},
		}
	}
	var res *pdv3.ProjectBatch
	res, err = s.v3.GetProjectBatch(ctx, &pdv3.GetProjectBatchRequest{
		Requests:  reqs,
		PageToken: pageToken,
	})
	if err != nil {
		return
	}
	out = res.Responses
	if pageToken != "" {
		var ls []*pdv3.ProjectBatch_Response
		ls, err = s.GetProjectsBatch(ctx, pageToken, in)
		if err != nil {
			return
		}
		out = append(out, ls...)
	}
	return
}

func ToSystem(system string) pb.System {
	system = strings.ToLower(system)
	switch system {
	case "npm", "javascript":
		return depsdev.System[purl.TypeNPM]
	case "cargo", "rust":
		return depsdev.System[purl.TypeCargo]
	case "pypi", "python":
		return depsdev.System[purl.TypePyPi]
	case "golang", "go":
		return depsdev.System[purl.TypeGolang]
	case "nuget":
		return depsdev.System[purl.TypeNuget]
	case "maven", "java":
		return depsdev.System[purl.TypeMaven]
	case "gem", "ruby":
		return depsdev.System[purl.TypeGem]
	default:
		return pb.System_SYSTEM_UNSPECIFIED
	}
}
