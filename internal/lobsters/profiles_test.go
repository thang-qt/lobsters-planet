package lobsters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchProfileParsesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><div class="labelled_grid">
<label>Joined</label><span><time datetime="2024-09-14 04:44:57">ago</time></span>
<label>Karma</label><span>38</span>
<label>Stories Submitted</label><span><a>3</a></span>
<label>Comments Posted</label><span><a>9</a></span>
<label>Homepage</label><span><a href="https://example.com/me/">site</a></span>
<label>About</label><div>Hello, world.</div>
</div></body></html>`))
	}))
	defer server.Close()

	profile, err := FetchProfile(context.Background(), server.Client(), server.URL+"/~person", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Username != "person" || profile.HomepageURL != "https://example.com/me/" {
		t.Fatalf("unexpected identity: %#v", profile)
	}
	if profile.JoinedAt == nil || profile.JoinedAt.Year() != 2024 {
		t.Fatalf("joined = %#v", profile.JoinedAt)
	}
	if profile.Karma == nil || *profile.Karma != 38 || profile.StoriesSubmitted == nil || *profile.StoriesSubmitted != 3 || profile.CommentsPosted == nil || *profile.CommentsPosted != 9 {
		t.Fatalf("unexpected counts: %#v", profile)
	}
	if profile.About != "Hello, world." {
		t.Fatalf("about = %q", profile.About)
	}
}

func TestFetchProfileWithoutHomepage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><label>About</label><span>hello</span></body></html>`))
	}))
	defer server.Close()

	profile, err := FetchProfile(context.Background(), server.Client(), server.URL+"/~person", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if profile.HomepageURL != "" {
		t.Fatalf("homepage = %q, want empty", profile.HomepageURL)
	}
}
