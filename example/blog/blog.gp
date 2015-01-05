package blog

type User table {
  ID int
  Name string
  Email string
  PermissionLevel int
  CryptPassword []byte

  index {
    Email
  }
}
