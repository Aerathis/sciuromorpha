package sciuromorpha

import (
	"errors"
	"io/ioutil"
	"os"
	dirpath "path"
	"strings"

	git "gopkg.in/libgit2/git2go.v24"
)

var checkoutOpts = &git.CheckoutOpts{
	Strategy: git.CheckoutSafe | git.CheckoutRecreateMissing | git.CheckoutAllowConflicts | git.CheckoutUseTheirs,
}

// Fetcher is an interface describing a remote fetcher
type Fetcher interface {
	Fetch([]string, *git.FetchOptions, string) error
	Free()
}

// Gitter is an interface representing the required operations for this library that a repository must implement
type Gitter interface {
	Free()
	RemotesLookup(string) (Fetcher, error)
	GetTag(string) (*git.Tag, error)
	CheckoutTree(*git.Tag, string, *git.CheckoutOpts) error
}

// GitClient manages a reference to a git repository on disk
type GitClient struct {
	repository Gitter
	repoPath   string
	sshPath    string
}

type gitterImpl struct {
	r *git.Repository
}

func (g *gitterImpl) Free() {
	g.r.Free()
}

func (g *gitterImpl) RemotesLookup(n string) (Fetcher, error) {
	return g.r.Remotes.Lookup(n)
}

func (g *gitterImpl) GetTag(tag string) (*git.Tag, error) {
	odb, err := g.r.Odb()
	if err != nil {
		return nil, err
	}
	defer odb.Free()

	var t *git.Tag
	odb.ForEach(func(oid *git.Oid) error {
		obj, err := g.r.Lookup(oid)
		if err != nil {
			return err
		}
		tObj, err := obj.AsTag()
		if err == nil {
			if tObj.Name() == tag {
				t = tObj
			}
		}
		return nil
	})
	return t, err
}

func (g *gitterImpl) CheckoutTree(t *git.Tag, tag string, o *git.CheckoutOpts) error {
	tagCommit, err := t.Target().AsCommit()
	if err != nil {
		return err
	}
	defer tagCommit.Free()

	tree, err := tagCommit.Tree()
	if err != nil {
		return err
	}
	defer tree.Free()

	err = g.r.CheckoutTree(tree, checkoutOpts)
	if err != nil {
		return err
	}

	err = g.r.SetHead("refs/tags/" + tag)
	if err != nil {
		return err
	}
	return nil
}

// OpenRepository opens a reference to a git repository at the given path
func OpenRepository(path, sshpath string) (gc *GitClient, err error) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return nil, err
	}
	gc = &GitClient{}
	gi := &gitterImpl{repo}
	gc.repository = gi
	gc.repoPath = path
	gc.sshPath = sshpath
	return
}

// Free ensures that resources held by the git client are properly freed
func (gc *GitClient) Free() {
	gc.repository.Free()
}

func getFetchOpts(gc *GitClient) *git.FetchOptions {
	return &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CertificateCheckCallback: func(*git.Certificate, bool, string) git.ErrorCode {
				return git.ErrOk
			},
			CredentialsCallback: func(string, string, git.CredType) (git.ErrorCode, *git.Cred) {
				ret, cred := git.NewCredSshKey("git", dirpath.Join(gc.sshPath, "id_rsa.pub"), dirpath.Join(gc.sshPath, "id_rsa"), "")
				return git.ErrorCode(ret), &cred
			},
		},
	}
}

func getFileInfoByName(prefix, name string) (os.FileInfo, error) {
	finfo, err := ioutil.ReadDir(prefix)
	if err != nil {
		return nil, err
	}
	for _, v := range finfo {
		if v.Name() == name {
			return v, nil
		}
	}
	return nil, errors.New("ERRNF")
}

type sparseEntries []string

func (se sparseEntries) contains(i string) bool {
	for _, v := range se {
		if v == i || v == i+string(os.PathSeparator) {
			return true
		}
	}
	return false
}

func isHidden(i string) bool {
	return i[0] == '.'
}

// CheckoutTag instructs the git client to checkout the provided tag onto disk from the repository
func (gc *GitClient) CheckoutTag(tag string) (err error) {
	r, err := gc.repository.RemotesLookup("origin")
	if err != nil {
		return err
	}
	defer r.Free()

	err = r.Fetch([]string{"+refs/heads/*:refs/remotes/origin/*", "refs/tags/*:refs/tags/*"}, getFetchOpts(gc), "")
	if err != nil {
		return err
	}

	t, err := gc.repository.GetTag(tag)
	if t != nil && err == nil {
		defer t.Free()
	} else {
		return errors.New("Unabled to find specified tag")
	}

	err = gc.repository.CheckoutTree(t, tag, checkoutOpts)
	if err != nil {
		return err
	}

	workPath := gc.repoPath
	g, err := getFileInfoByName(workPath, ".git")
	if err != nil {
		return err
	}

	sparseFlag := true
	workPath = dirpath.Join(workPath, g.Name())
	info, err := getFileInfoByName(workPath, "info")
	if err != nil {
		if err.Error() != "ERRNF" {
			return err
		}
		sparseFlag = false
	}

	if sparseFlag {
		workPath = dirpath.Join(workPath, info.Name())

		var sparse os.FileInfo
		sparse, err = getFileInfoByName(workPath, "sparse-checkout")
		if err != nil {
			if err.Error() != "ERRNF" {
				return err
			}
			sparseFlag = false
		}
		if sparse != nil {
			workPath = dirpath.Join(workPath, sparse.Name())
			sparseData, err := ioutil.ReadFile(workPath)
			if err != nil {
				return err
			}
			sparses := sparseEntries(strings.Split(string(sparseData), "\n"))
			dirContents, err := ioutil.ReadDir(gc.repoPath)
			if err != nil {
				return err
			}

			for _, v := range dirContents {
				if !sparses.contains(v.Name()) && !isHidden(v.Name()) {
					err = os.RemoveAll(dirpath.Join(gc.repoPath, v.Name()))
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
