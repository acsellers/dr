package forum

type User table {
	ID int
	FirstName, LastName string
  ScreenName string
  Email, password string
	BanExpiration *time.Time

  Posts []Post `column:"AuthorID"`
  Threads []Thread `column:"AuthorID"`
  postLikes []postLike
  LikedPosts []Post `through:"postLikes"`
  Blather Blather
  Timestamps
}

type Blather table {
  ID int
  UserID int

	Website string
	Signature string `type:"text"`
  Likes string `type:"text"`
  Dislikes string `type:"text"`
  Story string `type:"text"`
  AnswerA string
  AnswerB string
  AnswerC string
  AnswerD string
}

type Forum table {
	ID int
	Name string

  ForumBlather

	mods []forumMod
	Moderators []User `through:"mods"`
	pinneds []pinnedThread
	PinnedThreads []Thread `through:"pinneds"`
}

type ForumBlather subrecord {
	Rules string `type:"text"`
  Summary string `type:"text"`
  Events string `type:"text"`
  ImageURL string
  CSS string `type:"text"`
}

type forumMod table {
	ID int
	ForumID int
	UserID int `child:"User"`
}

type pinnedThread table {
	ID int
	ForumID int
	ThreadID int `child:"Thread"`
}

type Thread table {
	ID int
	Title string
	AuthorID int
	Author User
  Posts []Post
	Locked bool
}

type Post table {
	ID int
	ThreadID int
	Thread Thread
	Number int
	AuthorID int
	Author User
	ParentID *int
	Parent *Post

	Body string `type:"text"`
	likes []postLike
	Likers []User `through:"likes"`
}

type postLike table {
	ID int
	PostID int
	UserID int
}

type Timestamps mixin {
  CreatedAt, UpdatedAt time.Time
}

