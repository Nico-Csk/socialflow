// Package projecttest provides versionable automated checks for repository
// hygiene and project conventions. The tests in this package assert that
// documented Go tooling patterns and config files comply with the hardening
// rules defined in the socialflow specification.
//
// These tests serve as a regression gate: if a future change reintroduces
// ./... wildcards, removes -coverpkg from coverage commands, or violates
// .gitignore hygiene, the test suite catches it before code review.
package projecttest
