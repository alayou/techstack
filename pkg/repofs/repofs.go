package repofs

import (
	"context"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	scalibr "github.com/google/osv-scalibr"
	"github.com/google/osv-scalibr/binary/proto/config_go_proto"
	"github.com/google/osv-scalibr/extractor"
	scalibrfs "github.com/google/osv-scalibr/fs"
	pl "github.com/google/osv-scalibr/plugin/list"
	"github.com/hashicorp/go-version"
	"github.com/spf13/afero"
)

func isBinary(f *object.File) bool {
	is, _ := f.IsBinary()
	return is
}

// gitToMemFS 把 go-git 内存仓库转为 afero.MemMapFs
func gitToMemFS(repo *git.Repository, commitHash plumbing.Hash) (afero.Fs, error) {
	commit, err := repo.CommitObject(commitHash)
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	fs := afero.NewMemMapFs()
	// 遍历 Git tree 写入内存 FS
	return fs, tree.Files().ForEach(func(f *object.File) error {
		// 跳过非文本/大文件（优化）
		if f.Size > 1024*1024 || isBinary(f) {
			return nil
		}
		// 创建文件并写入内容
		file, err := fs.Create(f.Name)
		if err != nil {
			return err
		}
		defer file.Close()

		r, err := f.Reader()
		if err != nil {
			return err
		}
		defer r.Close()

		_, err = io.Copy(file, r)
		return err
	})
}

func GitCloneFromRemoteToFs(ctx context.Context, repoURL, ref string) (afero.Fs, string, error) {
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(ref),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, "", err
	}

	// 获取 HEAD commit
	head, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		return nil, "", err
	}
	commitHash := head.Hash()
	gitFs, err := gitToMemFS(repo, head.Hash())
	if err != nil {
		return nil, "", err
	}
	return gitFs, commitHash.String(), nil
}

// ScanGitRepo
func ScanGitRepo(ctx context.Context, repoURL, ref string) (*scalibr.ScanResult, error) {
	fs, _, err := GitCloneFromRemoteToFs(ctx, repoURL, ref)
	if err != nil {
		return nil, err
	}
	return ScanFromFs(ctx, afero.NewIOFS(fs)), nil
}

func ScanFromFs(ctx context.Context, fs scalibrfs.FS) *scalibr.ScanResult {
	plugins, _ := pl.FromNames([]string{"go", "python", "javascript", "rust"}, &config_go_proto.PluginConfig{
		DisableGoogleAuth: true,
	})
	cfg := &scalibr.ScanConfig{
		ScanRoots: []*scalibrfs.ScanRoot{{
			FS:   fs,
			Path: "/",
		}},
		Plugins:      plugins,
		UseGitignore: true,
		DirsToSkip:   []string{".git", "node_modules", "dist", "target"},
	}
	return scalibr.New().Scan(ctx, cfg)
}

func GitCloneFromRemoteToOsFs(ctx context.Context, repoURL, ref, dst string) error {
	gitFs, _, err := GitCloneFromRemoteToFs(ctx, repoURL, ref)
	if err != nil {
		return err
	}
	return copyDir(afero.NewIOFS(gitFs), "/", dst)
}

// GenWikiByGitnexus 使用Gitnexus命令行，生成wiki，
func GenWikiByGitnexus(ctx context.Context) {

}

func VersionFormat(ls []*extractor.Package) []*extractor.Package {
	for i, v := range ls {
		v1, err := version.NewVersion(v.Version)
		if err != nil {
			continue
		}
		v.Version = v1.String()
		ls[i] = v
	}
	return ls
}
