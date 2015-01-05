package blog

import (
	"log"
	"os"
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
		Log:        log.New(os.Stdout, "Migrate: ", 0),
	}
	err = db.Migrate()
	if err != nil {
		panic(err)
	}
	return c
}
