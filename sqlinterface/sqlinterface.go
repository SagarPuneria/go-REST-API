package SqlInterface

import (
	"database/sql"
	"sync"

	// Register mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// MySqldb contains db and dbRWMutex
type MySqldb struct {
	db        *sql.DB
	dbRWMutex sync.RWMutex
}

// CreateDataBase login to mysql db, create db, table if not exists
func CreateDataBase(DNS string, queries ...string) (*MySqldb, error) {
	var dbConn = new(MySqldb)
	dbConn.dbRWMutex.Lock()
	defer dbConn.dbRWMutex.Unlock()

	var Error error
	dbConn.db, Error = sql.Open("mysql", DNS)

	if Error != nil {
		return nil, Error
	}
	for _, query := range queries {
		_, err := dbConn.db.Exec(query)

		if err != nil {
			dbConn.Close()
			return dbConn, err
		}
	}
	return dbConn, nil
}

// Close the db
func (DBObject *MySqldb) Close() {
	DBObject.db.Close()
}

// ExecuteQuery excute the given query
func (DBObject *MySqldb) ExecuteQuery(strQuery string) error {
	DBObject.dbRWMutex.Lock()
	defer DBObject.dbRWMutex.Unlock()

	_, err := DBObject.db.Exec(strQuery)
	return err
}

// SelectQueryRow execute select query
func (DBObject *MySqldb) SelectQueryRow(strQuery string, TableID int) (int, error) {
	DBObject.dbRWMutex.Lock()
	defer DBObject.dbRWMutex.Unlock()
	var ID int
	err := DBObject.db.QueryRow(strQuery, TableID).Scan(&ID)

	return ID, err
}
