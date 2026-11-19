<?php
// Creates the SPMCIL AM (IT) course in the local MindMantra database, reusing the IIT Kanpur subjects (and so their questions) that overlap.
// Usage: php tools/mm-add-spmcil.php <user_id_to_enrol> [--dry-run]
foreach (file('D:/mindmantra.live/.env.local', FILE_IGNORE_NEW_LINES) as $l) {
  if (str_contains($l, '=') && !str_starts_with(trim($l), '#')) { [$k, $v] = explode('=', $l, 2); $env[trim($k)] = trim($v); }
}
$pdo = new PDO("mysql:host={$env['MYSQL_HOST']};port={$env['MYSQL_PORT']};dbname={$env['MYSQL_DATABASE']};charset=utf8mb4", $env['MYSQL_USER'], $env['MYSQL_PASSWORD'], [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]);
$userId = $argv[1] ?? '';
$dry = in_array('--dry-run', $argv, true);
if ($userId === '' || !$pdo->query("SELECT id FROM users WHERE id = " . (int)$userId)->fetch()) exit("unknown user id\n");

$slug = 'spmcil-am-it-2026';
if ($pdo->query("SELECT id FROM courses WHERE slug = " . $pdo->quote($slug))->fetch()) exit("course $slug already exists; nothing done\n");

$codes = ['iitk-computer-fundamentals', 'iitk-dbms-and-mysql', 'iitk-networks', 'iitk-operating-systems-and-linux', 'iitk-c-and-data-structures', 'iitk-python', 'iitk-security', 'iitk-web',
  'iitk-quantitative-aptitude', 'iitk-reasoning', 'iitk-english', 'iitk-current-affairs-and-gk'];

$pdo->beginTransaction();
try {
  $pdo->prepare("INSERT INTO courses (slug, title, description, time_limit, total_marks, total_questions, negative_marking, is_published, category) VALUES (?, ?, ?, 120, 150, 0, 0.25, 1, 'technical')")
    ->execute([$slug, 'SPMCIL AM (IT) 2026', 'Assistant Manager (IT), SPMCIL: 150 marks in 115 minutes plus 5 minutes of review. Professional Knowledge 90 marks (DBMS, networks, operating systems, data structures, OOP, security, software engineering, web), then general awareness, English, reasoning and quantitative aptitude 15 marks each. Practice draws on the subjects shared with the IIT Kanpur course.']);
  $courseId = (int)$pdo->lastInsertId();
  $link = $pdo->prepare("INSERT INTO course_subjects (course_id, subject_id, order_index) VALUES (?, ?, ?)");
  $total = 0;
  foreach ($codes as $i => $code) {
    $s = $pdo->query("SELECT id, name FROM subjects WHERE subject_code = " . $pdo->quote($code))->fetch();
    if (!$s) throw new Exception("subject missing: $code");
    $link->execute([$courseId, $s['id'], $i]);
    $n = (int)$pdo->query("SELECT COUNT(*) FROM questions WHERE is_active = 1 AND subject_id = " . (int)$s['id'])->fetchColumn();
    $total += $n;
    echo "  {$s['name']}: $n questions\n";
  }
  $pdo->prepare("UPDATE courses SET total_questions = ? WHERE id = ?")->execute([$total, $courseId]);
  $pdo->prepare("INSERT INTO course_enrollments (user_id, course_id) VALUES (?, ?)")->execute([(string)$userId, (string)$courseId]);
  echo "course $courseId ($slug), " . count($codes) . " subjects, $total questions in the practice pool, user $userId enrolled\n";
  if ($dry) { $pdo->rollBack(); echo "dry run: rolled back\n"; } else { $pdo->commit(); echo "committed\n"; }
} catch (Throwable $e) {
  $pdo->rollBack();
  echo "rolled back: ", $e->getMessage(), "\n";
  exit(1);
}
