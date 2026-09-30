// Package audit owns the append-only audit log. Every sensitive change produces an audit record.
//
// Part of the EduPilot modular monolith. Dependencies point inward only:
// presentation -> application -> domain -> ports (implemented by infrastructure).
package audit
