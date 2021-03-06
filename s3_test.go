// +build s3

package main

import (
	"context"
	. "github.com/aandryashin/matchers"
	"github.com/aerokube/selenoid/session"
	"github.com/aerokube/selenoid/upload"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	s3Srv *httptest.Server
)

func init() {
	s3Srv = httptest.NewServer(s3Mux())
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if strings.Contains(addr, "s3-mock.example.com") {
			addr = s3Srv.Listener.Addr().String()
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

func s3Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(_ http.ResponseWriter, _ *http.Request) {})
	return mux
}

var testSession = &session.Session{
	Quota: "some-user",
	Caps: session.Caps{
		Name:     "internet explorer",
		Version:  "11",
		Platform: "WINDOWS",
	},
}

func TestS3Uploader(t *testing.T) {
	uploader := &upload.S3Uploader{
		Endpoint:          "http://s3-mock.example.com",
		Region:            "us-west-1",
		AccessKey:         "some-access-key",
		SecretKey:         "some-secret-key",
		BucketName:        "test-bucket",
		KeyPattern:        "$fileName",
		ReducedRedundancy: true,
	}
	uploader.Init()
	f, _ := ioutil.TempFile("", "some-file")
	input := &upload.UploadRequest{
		Filename:  f.Name(),
		SessionId: "some-session-id",
		Session:   testSession,
		Type:      "log",
		RequestId: 4342,
	}
	uploaded, err := uploader.Upload(input)
	AssertThat(t, err, Is{nil})
	AssertThat(t, uploaded, Is{true})
}

func TestGetKey(t *testing.T) {
	const testPattern = "$quota/$sessionId_$browserName_$browserVersion_$platformName/$fileType$fileExtension"
	input := &upload.UploadRequest{
		Filename:  "/path/to/some-file.txt",
		SessionId: "some-session-id",
		Session:   testSession,
		Type:      "log",
		RequestId: 12345,
	}

	key := upload.GetS3Key(testPattern, input)
	AssertThat(t, key, EqualTo{"some-user/some-session-id_internet-explorer_11_windows/log.txt"})

	input.Session.Caps.S3KeyPattern = "$quota/$fileType$fileExtension"
	key = upload.GetS3Key(testPattern, input)
	AssertThat(t, key, EqualTo{"some-user/log.txt"})
}

func TestFileMatches(t *testing.T) {
	matches, err := upload.FileMatches("", "", "any-file-name")
	AssertThat(t, err, Is{nil})
	AssertThat(t, matches, Is{true})

	matches, err = upload.FileMatches("[", "", "/path/to/file.mp4")
	AssertThat(t, err, Not{nil})
	AssertThat(t, matches, Is{false})

	matches, err = upload.FileMatches("", "[", "/path/to/file.mp4")
	AssertThat(t, err, Not{nil})
	AssertThat(t, matches, Is{false})

	matches, err = upload.FileMatches("*.mp4", "", "/path/to/file.mp4")
	AssertThat(t, err, Is{nil})
	AssertThat(t, matches, Is{true})

	matches, err = upload.FileMatches("*.mp4", "", "/path/to/file.log")
	AssertThat(t, err, Is{nil})
	AssertThat(t, matches, Is{false})

	matches, err = upload.FileMatches("*.mp4", "", "/path/to/file.log")
	AssertThat(t, err, Is{nil})
	AssertThat(t, matches, Is{false})

	matches, err = upload.FileMatches("", "*.log", "/path/to/file.log")
	AssertThat(t, err, Is{nil})
	AssertThat(t, matches, Is{false})
}
