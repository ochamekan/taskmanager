package mysql

import (
	"context"
	"database/sql"
	"net"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"

	"github.com/example/taskmanager/migrations"
)

type ConnectionParams struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func (p ConnectionParams) DSN() string {
	cfg := mysqldriver.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(p.Host, p.Port)
	cfg.User = p.User
	cfg.Passwd = p.Password
	cfg.DBName = p.Database
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	return cfg.FormatDSN()
}

func Open(ctx context.Context, params ConnectionParams) (*sql.DB, error) {
	db, err := sql.Open("mysql", params.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	migrator, err := goose.NewProvider(goose.DialectMySQL, db, migrations.FS)
	if err != nil {
		return err
	}
	_, err = migrator.Up(ctx)
	return err
}
