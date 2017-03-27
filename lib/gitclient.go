package sciuromorpha

import (
	git "github.com/libgit2/git2go"
)

var checkoutOpts = &git.CheckoutOpts{
	Strategy: git.CheckoutSafe | git.CheckoutRecreateMissing | git.CheckoutAllowConflicts | git.CheckoutUseTheirs,
}

// GitClient manages a reference to a git repository on disk
type GitClient struct {
	repository *git.Repository
}

// OpenRepository opens a reference to a git repository at the given path
func OpenRepository(path string) (gc *GitClient, err error) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return nil, err
	}
	gc = &GitClient{}
	gc.repository = repo
	return
}

// Free ensures that resources held by the git client are properly freed
func (gc *GitClient) Free() {
	gc.repository.Free()
}

// CheckoutTag instructs the git client to checkout the provided tag onto disk from the repository
func (gc *GitClient) CheckoutTag(tag string) (err error) {
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
	defer t.Free()

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
	return
}
