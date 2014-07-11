package example

import (
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestUserSimple(t *testing.T) {
	c, err := Open("mysql", "root:toor@/doc_test")
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
	c, err := Open("mysql", "root:toor@/doc_test")
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
	c, err := Open("mysql", "root:toor@/doc_test")
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
