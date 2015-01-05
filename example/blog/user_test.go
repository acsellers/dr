package blog

import (
	"bytes"
	"log"
	"testing"

	"github.com/acsellers/dr/migrate"
	_ "github.com/mattn/go-sqlite3"
)

func TestUserSave(t *testing.T) {
	c := openTestConn()

	u := User{
		Name:                "Andrew",
		Email:               "andrew@example.com",
		PermissionLevel:     2,
		CryptPassword:       []byte("helloworld"),
		ArticleCompensation: 4.5,
		TotalCompensation:   1234.45,
	}
	err := u.Save(c)
	if err != nil {
		t.Fatal("User Save", err)
	}
	if u.ID == 0 {
		t.Fatal("User didn't update ID from result")
	}

	if c.User.Count() == 0 {
		t.Log(u)
		t.Fatal("User wasn't saved")
	}

	u2, err := c.User.Find(u.ID)
	if err != nil {
		t.Fatal("User Find", err)
	}
	if u2.Name != u.Name {
		t.Fatal("Name Compare", u.Name, u2.Name)
	}
	if u2.Email != u.Email {
		t.Fatal("Email Compare", u.Email, u2.Email)
	}
	if u2.PermissionLevel != u.PermissionLevel {
		t.Fatal("PermissionLevel Compare", u.PermissionLevel, u2.PermissionLevel)
	}
	if string(u2.CryptPassword) != string(u.CryptPassword) {
		t.Fatal("CryptPassword Compare", u.CryptPassword, u2.CryptPassword)
	}
	if u2.ArticleCompensation != u.ArticleCompensation {
		t.Fatal("ArticleCompensation Compare", u.ArticleCompensation, u2.ArticleCompensation)
	}
	if u2.TotalCompensation != u.TotalCompensation {
		t.Fatal("TotalCompensation Compare", u.TotalCompensation, u2.TotalCompensation)
	}

	u2, err = c.User.Name().Eq(u.Name).Retrieve()
	if err != nil {
		t.Fatal("User Name Retrieve", err)
	}
	if u2.Name != u.Name {
		t.Fatal("Name Compare", u.Name, u2.Name)
	}
	if u2.Email != u.Email {
		t.Fatal("Email Compare", u.Email, u2.Email)
	}
	if u2.PermissionLevel != u.PermissionLevel {
		t.Fatal("PermissionLevel Compare", u.PermissionLevel, u2.PermissionLevel)
	}
	if string(u2.CryptPassword) != string(u.CryptPassword) {
		t.Fatal("CryptPassword Compare", u.CryptPassword, u2.CryptPassword)
	}
	if u2.ArticleCompensation != u.ArticleCompensation {
		t.Fatal("ArticleCompensation Compare", u.ArticleCompensation, u2.ArticleCompensation)
	}
	if u2.TotalCompensation != u.TotalCompensation {
		t.Fatal("TotalCompensation Compare", u.TotalCompensation, u2.TotalCompensation)
	}

	u2, err = c.User.Email().Eq(u.Email).Retrieve()
	if err != nil {
		t.Fatal("User Email Retrieve", err)
	}
	if u2.Name != u.Name {
		t.Fatal("Name Compare", u.Name, u2.Name)
	}
	if u2.Email != u.Email {
		t.Fatal("Email Compare", u.Email, u2.Email)
	}
	if u2.PermissionLevel != u.PermissionLevel {
		t.Fatal("PermissionLevel Compare", u.PermissionLevel, u2.PermissionLevel)
	}
	if string(u2.CryptPassword) != string(u.CryptPassword) {
		t.Fatal("CryptPassword Compare", u.CryptPassword, u2.CryptPassword)
	}
	if u2.ArticleCompensation != u.ArticleCompensation {
		t.Fatal("ArticleCompensation Compare", u.ArticleCompensation, u2.ArticleCompensation)
	}
	if u2.TotalCompensation != u.TotalCompensation {
		t.Fatal("TotalCompensation Compare", u.TotalCompensation, u2.TotalCompensation)
	}

	c.Close()
}

func TestUserScopes(t *testing.T) {

	c := openTestConn()
	users := []User{
		User{
			Name:                "Hastur",
			Email:               "hastur@example.com",
			PermissionLevel:     1,
			CryptPassword:       []byte("asdf"),
			ArticleCompensation: 1.2,
			TotalCompensation:   2.4,
		},
		User{
			Name:                "Cthulhu",
			Email:               "cthulhu@example.com",
			PermissionLevel:     3,
			CryptPassword:       []byte("asdf"),
			ArticleCompensation: 1.8,
			TotalCompensation:   2.8,
		},
		User{
			Name:                "Yog-Sothoth",
			Email:               "yog-sothoth@example.com",
			PermissionLevel:     2,
			CryptPassword:       []byte("asdf"),
			ArticleCompensation: 1.8,
			TotalCompensation:   2.8,
		},
		User{
			Name:                "Tsathoggua",
			Email:               "tsathoggua@example.com",
			PermissionLevel:     2,
			CryptPassword:       []byte("asdf"),
			ArticleCompensation: 1.0,
			TotalCompensation:   2.0,
		},
		User{
			Name:                "Cthugha",
			Email:               "cthugha@example.com",
			PermissionLevel:     1,
			CryptPassword:       []byte("asdf"),
			ArticleCompensation: 1.2,
			TotalCompensation:   2.4,
		},
	}
	err := c.User.SaveAll(users)
	if err != nil {
		t.Fatal("User Save", err)
	}

	if c.User.Name().Eq("Cthulhu").Count() != 1 {
		t.Fatal("User not present")
	}

	if c.User.Name().Like("cthu%").Count() != 2 {
		t.Fatal("User.Like not working found:", c.User.Name().Like("cthu%").Count())
	}

	if c.User.Email().Like("%example.com").Count() != 5 {
		t.Fatal("Could not retrieve by email")
	}

	if c.User.PermissionLevel().Gt(1).Count() != 3 {
		t.Fatal("Could not find higher level users")
	}

	if c.User.ArticleCompensation().Gt(1).Count() != 4 {
		t.Fatal("Could not find highly compensated users")
	}
	if c.User.ArticleCompensation().Lte(1.21).Count() != 3 {
		t.Log(c.User.ArticleCompensation().Lte(1.21).QuerySQL())
		t.Fatal("Could not find cheaper users")
	}

	if c.User.TotalCompensation().Gt(2.5).Count() != 2 {
		t.Fatal("Could not find highly compensated users")
	}

	c.Close()
}

func openTestConn() *Conn {
	c, err := Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db := migrate.Database{
		DB:         c.DB,
		Schema:     Schema,
		Translator: NewAppConfig("sqlite3"),
		DBMS:       migrate.Sqlite,
		Log:        log.New(&bytes.Buffer{}, "Migrate: ", 0),
	}
	err = db.Migrate()
	if err != nil {
		panic(err)
	}
	return c
}
