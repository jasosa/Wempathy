package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	//needed for initialization
	_ "github.com/go-sql-driver/mysql"
	"github.com/jasosa/wemper/pkg/invitations"
	"strconv"
)

//Connection ...
type Connection interface {
	OpenConnection(stringConn string) (*sql.DB, error)
}

//PeopleSource mysql source for people entities
type PeopleSource struct {
	connection   Connection
	user         string
	password     string
	databasename string
	databasehost string
}

//NewPeopleSource creates a new instance of mysql people source
func NewPeopleSource(connection Connection, user, password, databasename, databasehost string) invitations.Source {
	pr := new(PeopleSource)
	pr.user = user
	pr.password = password
	pr.databasename = databasename
	pr.databasehost = databasehost
	pr.connection = connection
	return pr
}

// GetAllPeople gets all people from db source
func (pr PeopleSource) GetAllPeople() ([]invitations.AppUser, error) {

	db, err := openConnection(pr)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var query = "SELECT * from USERS"
	rows, err := db.Query(query)
	if err != nil {
		return nil, &SQLQueryError{Query: query, BaseError: err}
	}

	users := make([]invitations.AppUser, 0)
	for rows.Next() {
		var entryID int
		var name, email string
		var registered, admin bool
		err = rows.Scan(&entryID, &name, &email, &registered, &admin)
		if err != nil {
			return nil, &SQLQueryError{Query: query, BaseError: err}
		}
		users = append(users, pr.createUser(entryID, name, email, registered, admin))
	}

	return users, nil
}

//GetPersonByID gets the person with the specified id from the db source
func (pr PeopleSource) GetPersonByID(id string) (invitations.AppUser, error) {
	db, err := openConnection(pr)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var entryID int
	var name string
	var email string
	var registered bool
	var admin bool
	var query = "SELECT entryId, name, email, registered, admin FROM users WHERE entryID = ?"
	errScan := db.QueryRow(query, id).Scan(&entryID, &name, &email, &registered, &admin)

	if errScan != nil {
		return nil, &SQLQueryError{Query: query, BaseError: errScan}
	}

	appUser := pr.createUser(entryID, name, email, registered, admin)
	return appUser, nil
}

//AddPerson adds a new person to the db source
func (pr *PeopleSource) AddPerson(p invitations.AppUser) error {
	db, err := openConnection(*pr)
	if err != nil {
		return err
	}
	defer db.Close()

	var query = "INSERT INTO users (name, email,registered, admin) VALUES (?, ?, ?, ?)"
	result, errExec := db.Exec(query,
		p.GetPersonInfo().Name,
		p.GetPersonInfo().Email,
		p.GetPersonInfo().Registered,
		p.GetPersonInfo().Admin)

	if errExec != nil {
		return &SQLQueryError{Query: query, BaseError: errExec}
	}

	rowsAffected, errorRowsAff := result.RowsAffected()
	if errorRowsAff != nil {
		return &SQLQueryError{Query: query, BaseError: errorRowsAff}
	}

	if rowsAffected != 1 {
		return &SQLQueryError{Query: query, BaseError: errors.New("1 row affected was expected, but %s rows were affected")}
	}

	_, errorRLI := result.LastInsertId()
	if errorRLI != nil {
		return &SQLQueryError{Query: query, BaseError: errorRLI}
	}

	return nil
}

func (pr PeopleSource) createUser(entryID int, name, email string, registered, admin bool) invitations.AppUser {
	var appUser invitations.AppUser
	if admin {
		appUser = invitations.NewAdminUser(strconv.Itoa(entryID), name, email)
	} else if registered {
		appUser = invitations.NewRegisteredUser(strconv.Itoa(entryID), name, email)
	} else {
		appUser = invitations.NewInvitedUser(strconv.Itoa(entryID), name, email)
	}
	return appUser
}

func openConnection(pr PeopleSource) (*sql.DB, error) {
	stringCon := fmt.Sprintf("%s:%s@%s/%s", pr.user, pr.password, pr.databasehost, pr.databasename)
	db, err := pr.connection.OpenConnection(stringCon)
	if err != nil {
		return nil, &SQLOpeningDBError{BaseError: err}
	}
	return db, nil
}
