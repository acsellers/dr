package blog

type User table {
  ID                  int
  Name                string
  Email               string
  PermissionLevel     int
  CryptPassword       []byte
  ArticleCompensation float32
  TotalCompensation   float64
  Inactive            bool
  CreatedAt           time.Time

  index {
    Email
  }
}
