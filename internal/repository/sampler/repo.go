package sampler

import "i3_stat/internal/storage/database"

type Repo struct {
	db database.DBConnector
}

func InitRepo(db database.DBConnector) *Repo {
	return &Repo{db: db}
}
