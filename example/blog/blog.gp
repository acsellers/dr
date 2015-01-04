package blog

type User table {
  ID int
  Name string
  Email string
  CryptPassword string

  index {
    Email
  }
}
