package forum

import (
	"fmt"
	"log"
	"testing"

	"github.com/acsellers/doc/migrate"
	_ "github.com/mattn/go-sqlite3"
)

func TestUserSimple(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	sql, vals := c.User.ToSQL()
	if len(vals) > 0 {
		t.Fatal("Too many values in User.ToSQL")
	}
	if sql != "SELECT user.* FROM user" {
		t.Fatal("Incorrect sql", sql)
	}
	sql, vals = c.User.FirstName().Eq("Andrew").ToSQL()
	if len(vals) != 1 {
		t.Error(`Not correct values for []{"Andrew"}:`, vals)
	}
	if sql != "SELECT user.* FROM user WHERE user.firstname = ?" {
		t.Fatal("Incorrect SQL", sql)
	}

	sql, vals = c.User.ToSQL()
	if len(vals) > 0 || sql != "SELECT user.* FROM user" {
		t.Fatal("conn.User has been changed")
	}

	sql, vals = c.User.In(1, 2, 3, 4).ToSQL()
	if len(vals) != 4 {
		t.Error("Not correct values for []{1,2,3,4}")
	}
	if sql != "SELECT user.* FROM user WHERE user.id IN (?, ?, ?, ?)" {
		t.Fatal("SQL is incorrect for IN:", sql)
	}
	sql, vals = c.User.Between(1, 4).ToSQL()
	if len(vals) != 2 {
		t.Error("Not correct values for []{1,4}")
	}
	if sql != "SELECT user.* FROM user WHERE user.id BETWEEN ? AND ?" {
		t.Fatal("SQL is incorrect", sql)
	}

	sql, vals = c.User.Lte(4).Gte(1).ToSQL()
	if len(vals) != 2 {
		t.Error("Not correct values for []{4,1}")
	}
	if sql != "SELECT user.* FROM user WHERE user.id <= ? AND user.id >= ?" {
		t.Fatal("SQL is incorrect", sql)
	}

	sql, vals = c.User.Lt(5).Gt(0).ToSQL()
	if len(vals) != 2 {
		t.Error("Not correct values for []{5,0}")
	}
	if sql != "SELECT user.* FROM user WHERE user.id < ? AND user.id > ?" {
		t.Fatal("SQL is incorrect", sql)
	}

	sql, vals = c.User.CreatedAt().And(c.User.Eq(4)).ToSQL()
	if len(vals) != 1 {
		t.Error("Incorrect values for []{4}", vals)
	}
	if sql != "SELECT user.* FROM user WHERE user.id = ?" {
		t.Fatal("SQL is incorrect", sql)
	}

	sql, vals = c.User.Or(c.User.Eq(1), c.User.Eq(2), c.User.Or(c.User.Eq(3), c.User.Eq(4))).ToSQL()
	if len(vals) != 4 {
		t.Error("Incorrect values for []{1,2,3,4}:", vals)
	}
	if sql != "SELECT user.* FROM user WHERE (user.id = ? OR user.id = ? OR (user.id = ? OR user.id = ?))" {
		t.Fatal("SQL is incorrect", sql)
	}
}

func TestUserRetrieve(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	users, err := c.User.RetrieveAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatal("Different length of users, should be 3", fmt.Sprint(users))
	}
	u, err := c.User.LastName().Eq("Fellers").Retrieve()
	if err != nil {
		t.Fatal(err)
	}
	if u.FirstName != "Andrew" && u.LastName != "Fellers" {
		t.Fatal("Wrong user, expected a.fellers, got:", u)
	}

	u, err = c.User.Find(1)
	if err != nil {
		t.Fatal(err)
	}
	if u.FirstName != "Andrew" || u.LastName != "Sellers" {
		t.Fatal("Wrong user, expected a.fellers, got:", u)
	}
}

func TestUserCount(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	count := c.User.Count()
	if count != 3 {
		t.Fatal("Wrong number of users")
	}

	count = c.User.FirstName().CountOf()
	if count != 3 {
		t.Fatal("Wrong number of users")
	}

	count = c.User.FirstName().Distinct().CountOf()
	if count != 2 {
		t.Fatal("Wrong number of users")
	}
}

func TestPlucks(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	ids, err := c.User.PluckInt()
	if err != nil {
		t.Fatal("Couldn't pluck ids:", err)
	}
	if len(ids) != 3 {
		t.Fatal("Wrong number of users", ids)
	}

	names, err := c.User.FirstName().PluckString()
	if err != nil {
		t.Fatal("Couldn't pluck names:", err)
	}
	if len(names) != 3 {
		t.Fatal("Wrong number of users", names)
	}
	names, err = c.User.FirstName().Distinct().PluckString()
	if err != nil {
		t.Fatal("Couldn't pluck names:", err)
	}
	if len(names) != 2 {
		t.Fatal("Wrong number of users", names)
	}
	names, err = c.User.Eq(1).Pick("firstname || ' ' || lastname").PluckString()
	if err != nil {
		t.Fatal("Couldn't pluck names:", err)
	}
	if len(names) != 1 {
		t.Fatal("Wrong number of users", names)
	}
	if names[0] != "Andrew Sellers" {
		t.Fatal("Wrong name", names[0])
	}
}

func TestSaves(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	u := User{
		FirstName: "Ben",
		LastName:  "Smith",
	}
	err = u.Save(c)
	if err != nil {
		t.Fatal("Save Err:", err)
	}
	if u.ID == 0 {
		t.Fatal("Didn't update ID field on User")
	}

	count := c.User.LastName().Eq("Smith").Count()
	if count != 1 {
		t.Fatal("New user doesn't exist")
	}

	u.LastName = "Sisko"
	err = u.Save(c)
	if err != nil {
		t.Fatal("Save error:", err)
	}

	count = c.User.LastName().Eq("Smith").Count()
	if count != 0 {
		t.Fatal("New user doesn't exist")
	}

	count = c.User.LastName().Eq("Sisko").Count()
	if count != 1 {
		t.Fatal("User didn't get updated")
	}

	err = u.Delete(c)
	if err != nil {
		t.Fatal("Destroy Err:", err)
	}
}

func TestSetUpdate(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}

	err = c.User.FirstName().Eq("Nick").Set("Nicholas").Update()
	if err != nil {
		t.Fatal("Update error:", err)
	}

	countNo := c.User.FirstName().Eq("Nick").Count()
	countGood := c.User.FirstName().Eq("Nicholas").Count()
	if countNo != 0 || countGood != 1 {
		t.Fatal("Couldn't change user name with Set().Update()")
	}

	err = c.User.Eq(3).FirstName().Set("Nick").Update()
	if err != nil {
		t.Fatal("Update error:", err)
	}

	countGood = c.User.FirstName().Eq("Nick").Count()
	countNo = c.User.FirstName().Eq("Nicholas").Count()
	if countNo != 0 || countGood != 1 {
		t.Fatal("Couldn't change user name with Set().Update()")
	}

}

func TestJoins(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	b := Blather{
		Website: "http://nickiewickie.com",
	}
	u, err := c.User.FirstName().Eq("Nick").Retrieve()
	if err != nil {
		t.Fatal("Get user:", err)
	}
	b.UserID = u.ID
	b.Save(c)

	if cnt := c.Blather.Count(); cnt != 1 {
		t.Fatal("Incorrect blathers:", cnt)
	}

	scope := c.User.InnerJoin(Blathers)
	if cnt := scope.Count(); cnt != 1 {
		s, _ := scope.ToSQL()
		t.Fatal("Bad Count:", cnt, "Incorrect inner join:", s)
	}

	outcnt := c.User.OuterJoin(Blathers).Where("blather.id IS NULL").Count()
	if outcnt != 2 {
		t.Fatal("Incorrect outer join join")
	}
}

func TestSubrecord(t *testing.T) {
	c, err := OpenForTest("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Open:", err)
	}
	f := Forum{Name: "Lounge"}
	f.ForumBlather.Rules = "No Rules"
	err = f.Save(c)
	if err != nil {
		t.Fatal("Save")
	}

	cnt := c.Forum.ForumBlather().Rules().Eq("No Rules").Count()
	if cnt != 1 {
		t.Fatal("Count of subrecord fail")
	}

	if f.Rules != "No Rules" {
		fmt.Println(f)
	}

	f, err = c.Forum.ForumBlather().Include().Name().Eq("Lounge").Retrieve()
	if err != nil {
		t.Fatal("Subrecord retrive error:", err)
	}
	if f.Rules != "No Rules" {
		log.Println("Subrecord Retrieve Object:", f)
		log.Fatal("Cound no locate retrieved subrecord, expected '", "No Rules", "', found '", f.Rules, "'.")
	}
}

func OpenForTest(location, connection string) (*Conn, error) {
	c, err := Open(location, connection)
	if err != nil {
		return nil, err
	}
	db := migrate.Database{
		DB:         c.DB,
		Schema:     Schema,
		Translator: &AppConfig{},
		DBMS:       migrate.Generic,
	}
	db.Migrate()

	err = c.User.SaveAll([]User{
		User{FirstName: "Andrew", LastName: "Sellers"},
		User{FirstName: "Andrew", LastName: "Fellers"},
		User{FirstName: "Nick", LastName: "Sellers"},
	})

	return c, err
}
