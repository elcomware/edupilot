// Package authorization owns roles, permissions, scope and business-policy evaluation. Never duplicated per module.
//
// Part of the EduPilot modular monolith. Dependencies point inward only:
// presentation -> application -> domain -> ports (implemented by infrastructure).
package authorization
