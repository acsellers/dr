package forum

type User table {
	ID int
	FirstName, LastName string
        Email, password string
	Website string
	BanExpiration *time.Time
	Signature string `type:"text"`
  Threads []Thread `column:"AuthorID"`
  postLikes []postLike
  LikedPosts []Post `through:"postLikes"`
}

type Forum table {
	ID int
	Name string
	Rules string `type:"text"`
	mods []forumMod
	Moderators []User `through:"mods"`
	pinneds []pinnedThread
	PinnedThreads []Thread `through:"pinneds"`
}

type forumMod table {
	ID int
	ForumID int
	UserID int
}

type pinnedThread table {
	ID int
	ForumID int
	ThreadID int
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
