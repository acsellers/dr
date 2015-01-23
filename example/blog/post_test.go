package blog

import (
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestPostSave(t *testing.T) {
	c := openTestConn()

	users, err := createTestUsers(c)
	if err != nil {
		t.Fatal("User Save", err)
	}

	p, err := createSinglePost(c, users[0])
	if err != nil {
		t.Fatal("Post Save")
	}
	if p.ID == 0 {
		t.Fatal("Post ID")
	}

	si, se := p.Scope().PluckInt()
	ti, te := p.ToScope(c).PluckInt()
	if te != nil || se != nil {
		t.Fatal("Errors encoutered", se, te)
	}
	if len(si) != len(ti) || len(si) != 1 || si[0] != ti[0] {
		t.Fatal("Scope/ToScope isn't working", si, ti)
	}

	titles, err := p.Scope().Pick("LOWER(title)").PluckString()
	if err != nil {
		t.Fatal("Could not pick lower title")
	}
	if len(titles) != 1 || titles[0] != strings.ToLower(p.Title) {
		t.Fatal("Couldn't get lower title", titles)
	}

	if c.User.OuterJoin(c.Post).Count() != 5 {
		t.Fatal("Incorrect number of users")
	}
	if c.User.InnerJoin(c.Post).Count() != 1 {
		t.Log(c.User.InnerJoin(c.Post).QuerySQL())
		t.Fatal("Didn't INNER JOIN correctly")
	}

	createSinglePost(c, users[1])
	if c.User.InnerJoin(c.Post).Count() != 2 {
		t.Log(c.User.InnerJoin(c.Post).QuerySQL())
		t.Fatal("Didn't INNER JOIN correctly")
	}

	if c.User.OuterJoin(c.Post.Title().Eq(users[1].Name)).Count() != 1 {
		t.Log(c.User.OuterJoin(c.Post.Title().Eq(users[1].Name)).QuerySQL())
		t.Fatal("Didn't LEFT JOIN correctly")
	}

	posts, err := users[0].Post(c)
	if err != nil {
		t.Fatal("User.Post", err)
	}
	if len(posts) != 1 {
		t.Fatal("Can't use Post on User")
	}

	user, err := p.User(c)
	if err != nil {
		t.Fatal("Post.User", err)
	}
	if user.ID != p.UserID {
		t.Fatal("Post.User", user)
	}

	ps := users[0].Scope().PostScope()
	if ps.Count() != 1 {
		t.Fatal(ps.QuerySQL())
	}

	if users[0].Scope().SponsorScope().Count() != 0 {
		t.Fatal(users[0].Scope().SponsorScope().QuerySQL())
	}

	p.SponsorID = users[0].ID
	err = p.Save(c)
	if err != nil {
		t.Fatal("Post Save", err)
	}

	if users[0].Scope().SponsorScope().Count() != 1 {
		t.Fatal(users[0].Scope().SponsorScope().QuerySQL())
	}

	sponsored, err := users[0].Scope().SponsorScope().RetrieveAll()
	if err != nil {
		t.Fatal("Sponsor RetrieveAll", err)
	}
	if len(sponsored) != 1 || sponsored[0].Title != p.Title {
		t.Fatal(users[0].Scope().SponsorScope().QuerySQL())
	}

	sponsors, err := sponsored[0].Scope().SponsorScope().RetrieveAll()
	if err != nil {
		t.Fatal("Sponsor RetrieveAll", err)
	}
	if len(sponsors) != 1 || sponsors[0].ID != sponsored[0].SponsorID {
		t.Fatal(sponsors[0].Scope().SponsorScope().QuerySQL())
	}

	// end test
	c.Close()
}

func createSinglePost(c *Conn, u User) (Post, error) {
	p := Post{
		Title:  u.Name,
		Body:   "Post Body by ",
		UserID: u.ID,
	}

	return p, p.Save(c)
}
