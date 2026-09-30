// Package sync implements the durable outbox dispatcher, sync worker and inbound inbox with idempotency checks.
//
// Part of the EduPilot modular monolith. Dependencies point inward only:
// presentation -> application -> domain -> ports (implemented by infrastructure).
package sync
