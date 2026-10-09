CHANGES
=======
Version 0.1.0 - 2026-10-09
------------------
First version: the backend of Ships 3D, a 3D take on the Ships game where
players design ships from 3D primitives and fly them in open space.

- Users and sessions are shared with the 2D game (`users` and `sessions`
  collections, same `token` cookie), so one account and one login work in
  both. Registration, login (with "remember me"), logout and password change.
- 3D projects in their own `paintingProjects3d` collection, with a REST API
  (`/projects`) that only ever lets a user see or change their own projects,
  and server-side validation of every design.
- Real-time game over `/ws`: client-authoritative flight, server-authoritative
  life, kills, deaths and respawns. Reported hits are checked against the
  server's record of the bullet and the target before any damage applies.
- Built-in ships (Arrow, Hammer, Saucer) for guests and new players.
