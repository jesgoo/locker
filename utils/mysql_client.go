package utils

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type ExchangeItem struct {
	Amount     int    `json:"amount"`
	Method     int    `json:"method"`
	CreateTime string `json:"create_time"`
	Status     int    `json:"status"`
}

type DbClientStruct struct {
	location    string
	FirstAmount int
}

func (this *DbClientStruct) Init(location string, firstamount int) (err error) {
	this.location = location
	this.FirstAmount = firstamount
	return
}

func (this *DbClientStruct) SearchMoney(uid uint64) (money uint32, total uint32, err error) {
	money = 0
	db, err := sql.Open("mysql", this.location)
	if err != nil {
		//		WarningLog.Write("connect mysql fail .err[%s]", err.Error())
		WarningLog.Write("[SearchMoney] connect mysql fail . err[%s]", err.Error())
		return
	}
	defer db.Close()

	dataOut, err := db.Prepare("SELECT balance,total FROM user WHERE id = ?")
	if err != nil {
		//		WarningLog.Write("prepare sql fail . err[%s]", err.Error())
		WarningLog.Write("[SearchMoney] prepare sql fail . err[%s]", err.Error())
		return
	}
	defer dataOut.Close()
	err = dataOut.QueryRow(uid).Scan(&money, &total)
	if err != nil {
		//		WarningLog.Write("query user amount fail .err[%s]", err.Error())
		WarningLog.Write("[SearchMoney] query user amount fail .err[%s]", err.Error())
		return
	}
	DebugLog.Write("[SearchMoney] get money %d", money)
	return
}

func (this *DbClientStruct) SearchExchange(uid uint64) (ans []ExchangeItem, err error) {
	db, err := sql.Open("mysql", this.location)
	if err != nil {
		WarningLog.Write("[SearchExchange] connect mysql fail . err[%s]", err.Error())
		return
	}
	defer db.Close()
	sqlstr := fmt.Sprintf("SELECT * FROM cash WHERE user = %d", uid)
	rows, err := db.Query(sqlstr)
	if err != nil {
		WarningLog.Write("[SearchExchange] prepare sql fail . err[%s]", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var user uint64
		var amount int
		var method int
		var status int
		var create_time string
		var modified_time string
		err = rows.Scan(&id, &user, &amount, &method, &status, &create_time, &modified_time)
		if err != nil {
			WarningLog.Write("[SearchExchange] rows scanf fail . err[%s]", err.Error())
			return
		}
		var tmp ExchangeItem
		tmp.Amount = amount
		tmp.CreateTime = create_time
		tmp.Method = method
		tmp.Status = status
		ans = append(ans, tmp)
		DebugLog.Write("[SearchExchange] id [%d] amount[%d] create_time[%s]", id, amount, create_time)
	}
	return
}

func (this *DbClientStruct) InsertExchange(uid uint64, method int, amount int) (err error) {
	db, err := sql.Open("mysql", this.location)
	if err != nil {
		WarningLog.Write("[InsertExchange] connect mysql fail . err[%s]", err.Error())
		return
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		WarningLog.Write("[InsertExchange] start db.begin fail .err[%s]", err.Error())
		return
	}
	dataOut, err := tx.Prepare("SELECT balance FROM user WHERE id = ?")
	if err != nil {
		WarningLog.Write("[InsertExchange] prepare mysql fail. err[%s]", err.Error())
		tx.Rollback()
		return
	}
	var money uint64
	err = dataOut.QueryRow(uid).Scan(&money)
	if err != nil {
		WarningLog.Write("[InsertExchange] query uid money fail . err[%s]", err.Error())
		tx.Rollback()
		return
	}
	if money < uint64(amount) {
		WarningLog.Write("[InsertExchange] money is less than amout uid[%d] money[%d] amount[%d]", uid, money, amount)
		err = errors.New(" money is less than amout uid")
		tx.Rollback()
		return
	}
	dataOut, err = tx.Prepare("INSERT INTO cash SET user=? , amount=?, method=?, status=1")
	if err != nil {
		WarningLog.Write("[InsertExchange] prepare mysql fail. err[%s]", err.Error())
		tx.Rollback()
		return
	}
	_, err = dataOut.Exec(uid, amount, method)
	if err != nil {
		WarningLog.Write("[InsertExchange] insert exchange fail . err[%s]", err.Error())
		tx.Rollback()
		return
	}

	dataOut, err = tx.Prepare("UPDATE user SET balance = ? WHERE id = ?")
	if err != nil {
		WarningLog.Write("[InsertExchange] prepare sql fail, [%s]", err.Error())
		tx.Rollback()
		return
	}
	_, err = dataOut.Exec(money-uint64(amount), uid)
	if err != nil {
		WarningLog.Write("[InsertExchange] exec sql fail . err[%s]", err.Error())
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

func (this *DbClientStruct) AddUser(uid uint64) (ret int, err error) {
	ret = 404
	db, err := sql.Open("mysql", this.location)
	if err != nil {
		WarningLog.Write("[AddUser] connect mysql fail . err[%s]", err.Error())
		return
	}
	defer db.Close()
	sqlstr := fmt.Sprintf("SELECT * FROM user WHERE id = %d", uid)
	rows, err := db.Query(sqlstr)
	if err != nil {
		WarningLog.Write("[AddUser] sql fail . err[%s]", err.Error())
		return
	}
	if rows.Next() == true {
		WarningLog.Write("[AddUser] uid [%d] exist", uid)
		ret = 204
		return
	}
	DebugLog.Write("[AddUser] uid [%d] not exist\n", uid)
	dataOut, err := db.Prepare("INSERT INTO user SET id=?, balance=?, total=?")
	if err != nil {
		WarningLog.Write("[AddUser] prepare sql fail . err[%s]", err.Error())
		return
	}
	_, err = dataOut.Exec(uid, this.FirstAmount, this.FirstAmount)
	if err != nil {
		WarningLog.Write("[AddUser] insert a new user fail . err[%s]", err.Error())
		return
	}
	ret = 200
	return
}

var DbClient *DbClientStruct

/*
func main() {
	client := DbClient{}
	client.Init("root:root@tcp(192.168.2.5:3306)/locker?charset=utf8")
	client.SearchMoney(12345678901234)
	client.SearchExchange(123)
	client.InsertExchange(123333, 1, 20)
	client.AddUser(123)
	client.AddUser(1234)
}
*/
