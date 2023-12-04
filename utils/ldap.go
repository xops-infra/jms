package utils

import (
	"fmt"

	"github.com/go-ldap/ldap"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/noop/log"
)

type Ldap struct {
	Conn   *ldap.Conn
	Config config.Ldap
}

func NewLdap(config config.Ldap) *Ldap {
	log.Debugf("LDAP config: %+v", config)
	ldapConn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err != nil {
		fmt.Println(fmt.Sprintf("%s:%d", config.Host, config.Port))
		log.Panicf("Failed to connect to LDAP server: %s", err)
	}
	err = ldapConn.Bind(config.BindUser, config.BindPassword)
	if err != nil {
		log.Panicf("Bind to LDAP server failed: %s", err)
	}
	return &Ldap{
		Conn:   ldapConn,
		Config: config,
	}
}

func (l *Ldap) refreshLdap() error {
	ldapConn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", l.Config.Host, l.Config.Port))
	if err != nil {
		return fmt.Errorf("Failed to connect to LDAP server: %s", err)
	}
	err = ldapConn.Bind(l.Config.BindUser, l.Config.BindPassword)
	if err != nil {
		return fmt.Errorf("Bind to LDAP server failed: %s", err)
	}
	l.Conn = ldapConn
	return nil
}

func (l *Ldap) Login(username, password string) error {
	err := l.refreshLdap()
	if err != nil {
		return fmt.Errorf("Failed to refresh LDAP server: %s", err.Error())
	}
	searchRequest := ldap.NewSearchRequest(
		l.Config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(l.Config.UserSearchFilter, username), l.Config.Attributes,
		nil,
	)
	log.Debugf("searchRequest: %+v", searchRequest)
	sr, err := l.Conn.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("Failed to search LDAP server: %s", err.Error())
	}
	switch len(sr.Entries) {
	case 0:
		return fmt.Errorf("user %s not found", username)
	case 1:
		// Bind as the user to verify their password.
		err = l.Conn.Bind(sr.Entries[0].DN, password)
		if err != nil {
			log.Errorf("user %s login failed: %v", username, err)
			return fmt.Errorf("invalid password")
		} else {
			return nil
		}
	default:
		log.Errorf("ldap error, too many entries returned")
		return fmt.Errorf("too many entries returned")
	}
}
