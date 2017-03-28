package sciuromorpha

import (
	"errors"
	"io/ioutil"
	"os"
	dirpath "path"
	"strings"

	git "github.com/libgit2/git2go"
)

var checkoutOpts = &git.CheckoutOpts{
	Strategy: git.CheckoutSafe | git.CheckoutRecreateMissing | git.CheckoutAllowConflicts | git.CheckoutUseTheirs,
}

// GitClient manages a reference to a git repository on disk
type GitClient struct {
	repository *git.Repository
	repoPath   string
	sshPath    string
}

// OpenRepository opens a reference to a git repository at the given path
func OpenRepository(path, sshpath string) (gc *GitClient, err error) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return nil, err
	}
	gc = &GitClient{}
	gc.repository = repo
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
		if v == i {
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
	r, err := gc.repository.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	defer r.Free()

	err = r.Fetch([]string{"+refs/heads/*:refs/remotes/origin/*", "refs/tags/*:refs/tags/*"}, getFetchOpts(gc), "")
	if err != nil {
		return err
	}

	odb, err := gc.repository.Odb()
	if err != nil {
		return err
	}
	defer odb.Free()

	var t *git.Tag
	odb.ForEach(func(oid *git.Oid) error {
		obj, err := gc.repository.Lookup(oid)
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
	if t != nil {
		defer t.Free()
	} else {
		return errors.New("Unabled to find specified tag")
	}

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

	err = gc.repository.CheckoutTree(tree, checkoutOpts)
	if err != nil {
		return err
	}
	err = gc.repository.SetHead("refs/tags/" + tag)
	if err != nil {
		return err
	}

	workPath := gc.repoPath
	g, err := getFileInfoByName(workPath, ".git")
	if err != nil {
		return err
	}
	workPath = dirpath.Join(workPath, g.Name())
	info, err := getFileInfoByName(workPath, "info")
	if err != nil {
		return err
	}
	workPath = dirpath.Join(workPath, info.Name())

	sparseFlag := true
	sparse, err := getFileInfoByName(workPath, "sparse-checkout")
	if err != nil {
		if err.Error() != "ERRNF" {
			return err
		}
		sparseFlag = false
	}

	if sparseFlag {
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
				err = os.Remove(dirpath.Join(gc.repoPath, v.Name()))
				if err != nil {
					return err
				}
			}
		}
	}

	return
}
