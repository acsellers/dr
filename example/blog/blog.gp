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

  relation {
    []Post
  }

  index {
    Email
  }
}

type Post table {
  ID int
  Title string
  Body string `type:"text"`
  UserID int

  relation {
    User
  }

  index {
    UserID
  }
}
