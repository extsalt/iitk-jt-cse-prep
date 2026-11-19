<?php
// Read-only study report from the local MindMantra database.
// Usage: php tools/mm-report.php [user_id] [course_slug]   (defaults: 24, iitk-jt-cse-2026)
// DB credentials are read from D:/mindmantra.live/.env.local; nothing is written.
$userId = $argv[1] ?? '24';
$slug = $argv[2] ?? 'iitk-jt-cse-2026';
foreach (file('D:/mindmantra.live/.env.local', FILE_IGNORE_NEW_LINES) as $l) {
  if (str_contains($l, '=') && !str_starts_with(trim($l), '#')) { [$k, $v] = explode('=', $l, 2); $env[trim($k)] = trim($v); }
}
$pdo = new PDO("mysql:host={$env['MYSQL_HOST']};port={$env['MYSQL_PORT']};dbname={$env['MYSQL_DATABASE']};charset=utf8mb4", $env['MYSQL_USER'], $env['MYSQL_PASSWORD'], [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]);
$q = function (string $sql, array $p = []) use ($pdo) { $s = $pdo->prepare($sql); $s->execute($p); return $s->fetchAll(); };
$short = fn($s, $n = 70) => mb_strimwidth(preg_replace('/\s+/', ' ', (string)$s), 0, $n, '…');

$course = $q("SELECT id, title FROM courses WHERE slug = ?", [$slug])[0] ?? exit("no course $slug\n");
$tests = $q("SELECT id, title FROM courses WHERE slug LIKE ? ORDER BY slug", [$slug . '-mock-%']);
$testIds = array_column($tests, 'id');
$titles = array_column($tests, 'title', 'id');
$in = implode(',', array_map('intval', $testIds ?: [0]));
echo "== {$course['title']} (course {$course['id']}), user $userId\n\n== Exams\n";
$attempts = $q("SELECT * FROM cbt_attempts WHERE user_id = ? AND course_id IN ($in) ORDER BY completed_at", [$userId]);
foreach ($attempts as $a) {
  printf("%s  %-36s score %6.2f  C/W/S %d/%d/%d  time %d min%s\n", $a['completed_at'], $short($titles[$a['course_id']] ?? $a['course_id'], 36), $a['score'], $a['correct_count'], $a['incorrect_count'], $a['skipped_count'], intdiv((int)$a['time_taken_seconds'], 60), $a['status'] === 'completed' ? '' : "  [{$a['status']}]");
}
if (!$attempts) echo "none yet\n";

$outcomes = [];
foreach ($attempts as $a) {
  foreach (json_decode($a['answers_json'], true) ?: [] as $qid => $r) {
    if ($r['choice'] !== null) $outcomes[] = [$a['completed_at'], (int)$qid, (int)$r['choice'], $r['result'] === 'correct', 'exam'];
  }
}
foreach ($q("SELECT created_at, question_id, selected_option, is_correct, test_id FROM practice_attempts WHERE user_id = ? AND course_id = ? ORDER BY created_at, id", [$userId, $course['id']]) as $p) {
  $outcomes[] = [$p['created_at'], (int)$p['question_id'], (int)$p['selected_option'], (bool)$p['is_correct'], $p['test_id']];
}
usort($outcomes, fn($x, $y) => strcmp($x[0], $y[0]));

if ($attempts) {
  $last = end($attempts);
  echo "\n== Latest exam: " . ($titles[$last['course_id']] ?? '') . " ({$last['completed_at']})\nWeak topics (wrong + skipped):\n";
  $bd = json_decode($last['subject_breakdown_json'], true) ?: [];
  uasort($bd, fn($x, $y) => ($y['wrong'] + $y['skipped']) <=> ($x['wrong'] + $x['skipped']));
  foreach ($bd as $t => $c) if ($c['wrong'] + $c['skipped'] > 0) printf("  %-26s C %d  W %d  S %d\n", $t, $c['correct'], $c['wrong'], $c['skipped']);
  echo "Wrong answers:\n";
  $ans = json_decode($last['answers_json'], true) ?: [];
  $wrongIds = array_keys(array_filter($ans, fn($r) => $r['result'] === 'wrong'));
  if ($wrongIds) {
    foreach ($q("SELECT id, question_id, subject, question_text FROM questions WHERE id IN (" . implode(',', array_map('intval', $wrongIds)) . ")") as $w) {
      $r = $ans[$w['id']];
      printf("  %s [%s] chose %s, key %s | %s\n", $w['question_id'], $w['subject'], chr(65 + $r['choice']), chr(65 + $r['answer']), $short($w['question_text']));
    }
  }
}

$days = [];
foreach ($outcomes as $o) if ($o[4] !== 'exam') { $d = substr($o[0], 0, 10); $days[$d][0] = ($days[$d][0] ?? 0) + 1; $days[$d][1] = ($days[$d][1] ?? 0) + (int)$o[3]; }
echo "\n== Practice by day (answers, correct)\n";
foreach ($days as $d => [$n, $c]) echo "  $d  $n  $c\n";
if (!$days) echo "  none\n";

$qs = [];
foreach ($q("SELECT q.id, q.question_id, q.subject, q.correct_answer, q.question_text FROM questions q JOIN course_questions cq ON cq.question_id = q.id WHERE cq.course_id IN ($in)") as $r) $qs[(int)$r['id']] = $r;
$topic = []; $latest = []; $wrongCount = [];
foreach ($outcomes as [$when, $qid, $choice, $ok]) {
  if (!isset($qs[$qid])) continue;
  $t = $qs[$qid]['subject'];
  $topic[$t][0] = ($topic[$t][0] ?? 0) + 1;
  $topic[$t][1] = ($topic[$t][1] ?? 0) + (int)$ok;
  $latest[$qid] = [$ok, $choice, $when];
  if (!$ok) $wrongCount[$qid] = ($wrongCount[$qid] ?? 0) + 1;
}
echo "\n== Topic accuracy, exams and practice (answers, accuracy), weakest first\n";
uasort($topic, fn($x, $y) => ($x[1] / $x[0]) <=> ($y[1] / $y[0]));
foreach ($topic as $t => [$n, $c]) printf("  %-26s %3d  %5.1f%%\n", $t, $n, $c * 100 / $n);
$open = array_filter($latest, fn($l) => !$l[0]);
echo "\n== Open mistakes (latest answer wrong): " . count($open) . "\n";
uksort($open, fn($x, $y) => ($wrongCount[$y] ?? 0) <=> ($wrongCount[$x] ?? 0));
foreach ($open as $qid => [$ok, $choice, $when]) {
  $r = $qs[$qid];
  printf("  %s [%s] wrong %dx, last chose %s (key %s) %s | %s\n", $r['question_id'], $r['subject'], $wrongCount[$qid], chr(65 + $choice), chr(65 + (int)$r['correct_answer']), substr($when, 0, 16), $short($r['question_text'], 60));
}
