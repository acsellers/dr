package example

import "testing"

func TestUserSimple(t *testing.T) {
	c, _ := Open("blah")
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
		t.Error("Incorrect SQL", sql)
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
}
