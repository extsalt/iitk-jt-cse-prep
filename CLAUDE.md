# IIT Kanpur Junior Technician (CSE) exam prep

This folder holds mock tests and practical notes for Ganesh's IIT Kanpur Junior Technician (CSE) exam, Advt. No. 1/2026.

## The exam

- CBT on Sunday 29 Nov 2026, 2:30 pm to 5:00 pm, at IIT Kanpur.
- 120 marks in 2.5 hours:
  - Part A, 40 marks, objective: current affairs, quantitative aptitude, numerical ability, logical reasoning, analytical and verbal ability.
  - Part B, 60 marks, objective: CSE domain.
  - Part C, 20 marks, descriptive, English only: official letter, memo/note, report, case study, incident report.
- Parts A and B: +1 for a correct answer, -0.25 for a wrong one. Part C is evaluated only if the minimum is cleared in A and B.
- Final merit is on the written test only. The practical test (JOPT, 1 Dec 2026) is qualifying and covers the same syllabus.

## Part B syllabus

Theory:
- Computer fundamentals.
- C programming and data structures: arrays, pointers, stacks, queues, linked lists, trees, graphs, searching, sorting.
- Operating systems: Linux and Windows, process/memory/storage management, file systems, distributed systems, security.
- Networks: OSI, topologies, LAN/VLAN, subnetting and supernetting, TCP/UDP, Wi-Fi, basic security, SAN.
- DBMS: user and role management, queries, optimization, backup and restore, MySQL/MariaDB scripts.

Practical:
- Linux commands, bash scripting, users, permissions, file systems, socket programming, partitions, OS install, packages.
- LDAP, Git, VirtualBox, KVM, LXC, open-source cloud.
- Mail, web, file and print servers; backup; firewall.
- Cisco IOS basics, DNS, DHCP, VLAN, TFTP, SNMP, tcpdump.
- Hardware assembly and cable crimping.
- HTML5, CSS, JavaScript, Ajax, PHP.
- Python basics, OOP, data visualization, Django.

## Folder layout

- `mocks/mock-NN/`: one folder per test, with `mock-NN.tex`, `mock-NN-key.tex` and the compiled PDFs.
- `pyq/`: `pyq-analysis.tex` and `.pdf` (what past papers ask, source links, key errors found) and `raw/` (extracted text of the IITH papers).
- `notes/practical-notes.tex` and `.pdf`: the single notes file for the whole practical syllabus (add new topics to this file, not new files). Practical syllabus only. Same house style as the mocks; use the `\pyq{}` tag for points that came from a past paper, and `lstlisting` for code. Pure CS theory is studied with a separate system, so do not write theory notes here.

## How tests are built

- Full mock: 40 Part A + 60 Part B objective questions and 2 Part C descriptive questions, matching the marks split above.
- Also wanted: subject tests and section tests of 30 to 40 questions, aimed at weak areas.
- Every objective question has four options and exactly one correct answer. Spread the correct options evenly across (a) to (d).
- The key is a separate file, with a one-line reason for each answer. Verify every answer before writing it; technical accuracy matters more than volume.
- No real IIT Kanpur past paper was found. Style reference: the IIT Hyderabad 2025 Junior Technician (Computer Centre) and Technician Superintendent (CSE) papers. Mock 01 is baseline difficulty; raise difficulty in areas that score well.
- LaTeX: `pdflatex`, article class, slate (#2F3E4E) and ember (#C8501E) accent colours. Reuse the `\opts` and `\optl` macros from `mocks/mock-01/mock-01.tex`. Compile twice, then delete `.aux`, `.log`, `.out`.

## How Ganesh likes to work

- Honest, direct feedback; no generic encouragement.
- Intuition first, then diagrams and comparison tables. Keep explanations short.
- Everything he studies from is LaTeX compiled to PDF: mocks, keys, notes, analyses. No markdown deliverables (told on 2026-11-15). Chat replies stay plain text.
- New topics in small steps, one at a time.
- He already knows basic Linux user and file creation, CIDR-to-mask conversion, and bash basics (variables, arguments, if, for).
- Practice machine: an Ubuntu VM named `ubuntu-lab` in VirtualBox on this Windows host, reached over SSH.

## Progress log

- 2026-11-14: Mock Test 01 created. Not yet attempted.
- 2026-11-14: Searched for past papers. No IITK JT (CSE) paper is public; the IITK archive lists none. Read five IIT Hyderabad CSE/Computer Centre papers (2023, 2025); raw text in `pyq/raw/`, analysis in `pyq/pyq-analysis.md`. Mock Test 02 created from their question patterns (every key line is tagged with the source question). Mock 02 not yet attempted.
- 2026-11-15: Notes and PYQ analysis converted from markdown to LaTeX (`notes/practical-notes.tex`, `pyq/pyq-analysis.tex`, PDFs compiled); the markdown copies are deleted. The `.tex` files are now the source.
- 2026-11-15: All notes merged into one file, `notes/practical-notes`, and extended to cover the rest of the practical syllabus (permissions, disks, packages, Cisco IOS, DHCP/DNS/SNMP/TFTP, firewall, mail/file/print, LDAP, backup, KVM/LXC/OpenStack, Git, Python OOP, sockets, Django, MySQL, cabling). Examples run here: Git, awk, Python. The rest is from standard usage and still to be checked on `ubuntu-lab`.
