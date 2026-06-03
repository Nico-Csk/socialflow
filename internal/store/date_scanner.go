package store

import (
	"fmt"
	"time"
)

// dateScanner implements database/sql.Scanner to decode PostgreSQL DATE
// columns (OID 1082) into a *string formatted as YYYY-MM-DD.
//
// pgx v5 does not have a native scan plan for DATE→*string, so this
// adapter handles the binary time.Time and text-fallback cases.
type dateScanner struct {
	dest **string
}

// Scan satisfies the database/sql.Scanner interface. It decodes src
// (a pgx-decoded value: nil, time.Time, string, or []byte) into *dest
// as a YYYY-MM-DD string or nil.
func (d dateScanner) Scan(src any) error {
	if d.dest == nil {
		return fmt.Errorf("dateScanner: dest is nil")
	}

	switch v := src.(type) {
	case nil:
		*d.dest = nil
		return nil
	case time.Time:
		s := v.Format("2006-01-02")
		*d.dest = &s
		return nil
	case string:
		*d.dest = &v
		return nil
	case []byte:
		s := string(v)
		*d.dest = &s
		return nil
	default:
		return fmt.Errorf("dateScanner: cannot scan %T into *string", src)
	}
}
