package blog

type User table {
  ID                  int
  Name                string
  Email               string
  PermissionLevel     int
  ArticleCompensation float32
  TotalCompensation   float64
  Inactive            bool
  CreatedAt           time.Time

  SecurePassword

  relation {
    []Post
  }

  index {
    Email
  }
}

type SecurePassword mixin {
  CryptPassword []byte
}

func (sp *SecurePassword) SetPassword(password string) {
  sp.CryptPassword, err = bcrypt.GenerateFromPassword([]byte(password), 0)
  if err != nil {
    log.Println("SetPassword", err)
  }
}

func (sp SecurePassword) ComparePassword(password string) bool {
  if bcrypt.CompareHashAndPassword(sp.CryptPassword, []byte(password)) == nil {
    return true
  }
  return false
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
