// Package inventory owns products, variants, stock movements and stock counts. Quantity derives from movement history, never a mutable field.
//
// Part of the EduPilot modular monolith. Dependencies point inward only:
// presentation -> application -> domain -> ports (implemented by infrastructure).
package inventory
