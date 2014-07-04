package main

type User table {
  FirstName, LastName string `null="zero"`
  Timestamps
}

type Appointment table {
  ID int
  Timestamps
}

type Timestamps mixin {
  CreatedAt time.Time
  UpdatedAt time.Time
}

func (t Timestamps) HasBeenUpdated() bool {
  return t.UpdatedAt.After(t.CreatedAt)
}
