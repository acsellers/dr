package example

func (u User) Name() string {
	return u.FirstName + " " + u.LastName
}
