package flags

import (
	"flag"
)

var (
	flagRunAddr         string
	flagDatabaseURI     string
	flagAccrualAddress  string
)

func GetFlagRunAddr() string {
	return flagRunAddr
}

func SetFlagRunAddr(newFlagRunAddr string) {
	flagRunAddr = newFlagRunAddr
}

func GetFlagDatabaseURI() string {
	return flagDatabaseURI
}

func SetFlagDatabaseURI(newFlagDatabaseURI string) {
	flagDatabaseURI = newFlagDatabaseURI
}

func GetFlagAccrualAddress() string {
	return  flagAccrualAddress
}

func SetFlagAccrualAddress(newFlagAccrualAddress string) {
	flagAccrualAddress = newFlagAccrualAddress
}

func ParseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8081", "address and port to run server")
	flag.StringVar(&flagDatabaseURI, "d", "", "database connection string")
	flag.StringVar(&flagAccrualAddress, "r", "http://localhost:8080", "accrual system address")

	flag.Parse()
}
