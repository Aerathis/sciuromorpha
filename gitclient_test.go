package sciuromorpha

import (
	"os"
	"path"
	"strings"
	"testing"

	git "gopkg.in/libgit2/git2go.v24"
)

var se = sparseEntries([]string{"first", "second", "third"})
var testHook string

type testFetcher struct {
	Fail bool
}

func (tf testFetcher) Fetch([]string, *git.FetchOptions, string) error {
	return nil
}

func (tf testFetcher) Free() {
}

type testGitter struct {
	FailRemote bool
	FailFetch  bool
	MissingTag bool
	FailTree   bool
}

func (tg *testGitter) Free() {
	testHook = "Free called"
}

func (tg *testGitter) RemotesLookup(s string) (Fetcher, error) {
	return testFetcher{}, nil
}

func (tg *testGitter) GetTag(s string) (*git.Tag, error) {
	return &git.Tag{}, nil
}

func (tg *testGitter) CheckoutTree(*git.Tag, string, *git.CheckoutOpts) error {
	return nil
}

var tg = testGitter{}
var testClient = GitClient{
	repository: &tg,
	repoPath:   "",
	sshPath:    "",
}

func createLocalDir(name string) (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	result := path.Join(d, name)
	err = os.Mkdir(result, os.ModeDir|os.ModePerm)
	return result, err
}

func initRepo(path string) (*git.Repository, error) {
	return git.InitRepository(path, false)
}

func openRepo(path, sshpath string) (*GitClient, error) {
	p, err := createLocalDir(path)
	if err != nil {
		return nil, err
	}
	_, err = initRepo(p)
	if err != nil {
		return nil, err
	}
	return OpenRepository(path, sshpath, func(g *git.Repository) Gitter {
		return &testGitter{}
	})
}

func TestFree(t *testing.T) {
	testClient.Free()
	if testHook != "Free called" {
		t.Fail()
	}
}

func TestSparseEntriesDoesContain(t *testing.T) {
	if !se.contains("second") {
		t.Fail()
	}
}

func TestSparseEntriesDoesNotContain(t *testing.T) {
	if se.contains("fourth") {
		t.Fail()
	}
}

func TestIsHidden(t *testing.T) {
	if !isHidden(".hiddenDir") {
		t.Fail()
	}
}

func TestIsNotHidden(t *testing.T) {
	if isHidden("nothiddendir") {
		t.Fail()
	}
}

func TestGetFetchOptsCredentialsCallbackEmptySSHPath(t *testing.T) {
	opt := getFetchOpts(&testClient)
	gitErr, cred := opt.RemoteCallbacks.CredentialsCallback("", "", git.CredTypeSshKey)
	if gitErr != 0 {
		t.Fail()
	}
	if cred == nil {
		t.Fail()
	}
}

func TestGetFetchOptsCredentialsCallbackNonexistentCredentials(t *testing.T) {
	sshPath, err := createLocalDir(".ssh")
	defer os.RemoveAll(sshPath)
	if err != nil {
		t.Error(err)
	}

	testClient.sshPath = sshPath
	opt := getFetchOpts(&testClient)
	gitErr, cred := opt.RemoteCallbacks.CredentialsCallback("", "", git.CredTypeSshKey)
	// Strangely if the files don't exist this doesn't cause an error
	if gitErr != 0 {
		t.Fail()
	}
	if cred == nil {
		t.Fail()
	}
}

func TestGetFetchOptsCredentialsCallback(t *testing.T) {
	// Create files for testing
	sshPath, err := createLocalDir(".ssh")
	defer os.RemoveAll(sshPath)
	if err != nil {
		t.Error(err)
	}

	_, err = os.OpenFile(path.Join(sshPath, "id_rsa.pub"), os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}
	_, err = os.OpenFile(path.Join(sshPath, "id_rsa"), os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}
	testClient.sshPath = sshPath
	opt := getFetchOpts(&testClient)
	gitErr, cred := opt.RemoteCallbacks.CredentialsCallback("", "", git.CredTypeSshKey)
	if gitErr != 0 {
		t.Fail()
	}
	if cred == nil {
		t.Fail()
	}
}

func TestGetFetchOptsCertificateCheckCallback(t *testing.T) {
	opt := getFetchOpts(&testClient)
	gitErr := opt.RemoteCallbacks.CertificateCheckCallback(&git.Certificate{}, true, "")
	if gitErr != git.ErrOk {
		t.Fail()
	}
}

func TestCheckoutTagNoSparse(t *testing.T) {
	testClient, err := openRepo("testing", "")
	defer os.RemoveAll("testing")
	if err != nil {
		t.Error(err)
	}

	err = testClient.CheckoutTag("test")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestCheckoutTagNoGitDir(t *testing.T) {
	testDir, err := createLocalDir("testing")
	defer os.RemoveAll(testDir)
	if err != nil {
		t.Error(err)
	}

	testClient.repoPath = testDir

	err = testClient.CheckoutTag("test")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "ERRNF" {
		t.Fail()
	}
}

func TestCheckoutTagNoRepoPathSet(t *testing.T) {
	testDir, err := createLocalDir("testing")
	defer os.RemoveAll(testDir)
	if err != nil {
		t.Error(err)
	}

	err = testClient.CheckoutTag("test")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "ERRNF" {
		t.Fail()
	}
}

func TestCheckoutTagRepoPathNotExist(t *testing.T) {
	testClient.repoPath = "/directorycertainlydoesntexist"
	err := testClient.CheckoutTag("test")
	if err == nil {
		t.Fail()
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fail()
	}
}
