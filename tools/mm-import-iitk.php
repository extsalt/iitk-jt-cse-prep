<?php
// One-time import of runner/banks/mock-*.json into the local MindMantra database (run on 2026-11-18).
// It refuses to run if the course already exists; to add a new bank later, extend it to import only the missing mock courses.
// Usage: php tools/mm-import-iitk.php <user_id_to_enrol> [--dry-run]
foreach (file('D:/mindmantra.live/.env.local', FILE_IGNORE_NEW_LINES) as $l) {
  if (str_contains($l, '=') && !str_starts_with(trim($l), '#')) { [$k, $v] = explode('=', $l, 2); $env[trim($k)] = trim($v); }
}
$pdo = new PDO("mysql:host={$env['MYSQL_HOST']};port={$env['MYSQL_PORT']};dbname={$env['MYSQL_DATABASE']};charset=utf8mb4", $env['MYSQL_USER'], $env['MYSQL_PASSWORD'], [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]);
$userId = $argv[1] ?? '';
$dry = in_array('--dry-run', $argv, true);
if ($userId === '' || !$pdo->query("SELECT id FROM users WHERE id = " . (int)$userId)->fetch()) exit("unknown user id\n");

$slug = 'iitk-jt-cse-2026';
if ($pdo->query("SELECT id FROM courses WHERE slug = " . $pdo->quote($slug))->fetch()) exit("course $slug already exists; nothing done\n");

preg_match_all('/\{"([^"]+)", \[\]string\{([^}]*)\}\}/s', file_get_contents('D:/iit-kanpur-lab/runner/topics.go'), $m, PREG_SET_ORDER);
$areaOf = []; $areaOrder = [];
foreach ($m as $i => $row) {
  $areaOrder[$row[1]] = $i;
  preg_match_all('/"([^"]+)"/', $row[2], $tm);
  foreach ($tm[1] as $j => $t) $areaOf[$t] = [$row[1], $j];
}

$banks = [];
foreach (glob('D:/iit-kanpur-lab/runner/banks/mock-*.json') as $f) $banks[] = json_decode(file_get_contents($f), true);
usort($banks, fn($a, $b) => strcmp($a['id'], $b['id']));

$clean = fn(string $s) => str_replace(['**', '`'], '', $s);
$slugify = fn(string $s) => trim(preg_replace('/[^a-z0-9]+/', '-', strtolower($s)), '-');

$usedAreas = []; $usedTopics = [];
foreach ($banks as $b) foreach ($b['questions'] as $q) {
  if (!isset($areaOf[$q['topic']])) exit("topic not in topics.go: {$q['topic']}\n");
  [$area, $ord] = $areaOf[$q['topic']];
  $usedAreas[$area] = $areaOrder[$area];
  $usedTopics[$q['topic']] = [$area, $ord];
}
asort($usedAreas);

$pdo->beginTransaction();
try {
  $pdo->prepare("INSERT INTO courses (slug, title, description, time_limit, total_marks, total_questions, negative_marking, is_published, category) VALUES (?, ?, ?, 115, 100, ?, 0.25, 1, 'technical')")
    ->execute([$slug, 'IIT Kanpur JT (CSE) 2026', 'Junior Technician (CSE), Advt. No. 1/2026: four full mock tests (Parts A and B, 100 questions each, +1/−0.25) and topic-wise practice with explanations. Part C (letter, memo, report) is written on paper.', 100 * count($banks)]);
  $courseId = (int)$pdo->lastInsertId();

  $subjectId = []; $order = 0;
  $insSubject = $pdo->prepare("INSERT INTO subjects (subject_code, name, description, order_index) VALUES (?, ?, ?, ?)");
  $insCS = $pdo->prepare("INSERT INTO course_subjects (course_id, subject_id, order_index) VALUES (?, ?, ?)");
  foreach (array_keys($usedAreas) as $area) {
    $insSubject->execute(['iitk-' . $slugify($area), $area, 'IIT Kanpur JT (CSE): ' . $area, 100 + $order]);
    $subjectId[$area] = (int)$pdo->lastInsertId();
    $insCS->execute([$courseId, $subjectId[$area], $order++]);
  }
  $moduleId = [];
  $insModule = $pdo->prepare("INSERT INTO subject_modules (module_title, module_code, subject_id, order_index, status) VALUES (?, ?, ?, ?, 1)");
  uasort($usedTopics, fn($a, $b) => [$usedAreas[$a[0]], $a[1]] <=> [$usedAreas[$b[0]], $b[1]]);
  foreach ($usedTopics as $topic => [$area, $ord]) {
    $insModule->execute([$topic, 'iitk-' . $slugify($topic), $subjectId[$area], $ord]);
    $moduleId[$topic] = (int)$pdo->lastInsertId();
  }

  $insQ = $pdo->prepare("INSERT INTO questions (question_text, options, correct_answer, subject, subject_id, subject_module_id, year, quality_status, reviewed_by, reviewed_at, is_active, question_id, explanation, ai_review_status) VALUES (?, ?, ?, ?, ?, ?, 2026, 'verified', 'claude-verified', NOW(), 1, ?, ?, 'none')");
  $insTest = $pdo->prepare("INSERT INTO courses (slug, title, description, time_limit, total_marks, total_questions, negative_marking, is_published, category) VALUES (?, ?, ?, 115, 100, ?, 0.25, 0, 'mock_test')");
  $insCQ = $pdo->prepare("INSERT INTO course_questions (course_id, question_id, is_active) VALUES (?, ?, 1)");
  $newId = [];
  foreach ($banks as $b) {
    $insTest->execute([$slug . '-' . $b['id'], 'IITK JT (CSE) ' . $b['title'], 'Parts A and B of ' . $b['title'] . ': 40 general ability + 60 CSE questions in 115 minutes. Write Part C on paper.', count($b['questions'])]);
    $testId = (int)$pdo->lastInsertId();
    foreach ($b['questions'] as $q) {
      $text = (!empty($q['passage']) ? "Passage: " . $clean($q['passage']) . "\n\n" : '') . $clean($q['question']) . (!empty($q['code']) ? "\n\n" . $q['code'] : '');
      $exp = $clean($q['explanation']) . (!empty($q['source']) ? ' [Source: ' . $q['source'] . ']' : '');
      $insQ->execute([$text, json_encode(array_map($clean, $q['options']), JSON_UNESCAPED_UNICODE), ord($q['answer']) - 97, $q['topic'], $subjectId[$areaOf[$q['topic']][0]], $moduleId[$q['topic']], $q['id'], $exp]);
      $newId[$q['id']] = (int)$pdo->lastInsertId();
      $insCQ->execute([$testId, $newId[$q['id']]]);
    }
    echo "{$b['id']}: test course $testId, " . count($b['questions']) . " questions\n";
  }

  $pdo->prepare("INSERT INTO course_enrollments (user_id, course_id) VALUES (?, ?)")->execute([(string)$userId, (string)$courseId]);

  $imported = 0;
  $log = 'D:/iit-kanpur-lab/runner/data/practice-log.jsonl';
  if (is_file($log)) {
    $insPA = $pdo->prepare("INSERT INTO practice_attempts (user_id, test_id, question_id, course_id, selected_option, is_correct, time_spent_seconds, created_at) VALUES (?, 'runner-import', ?, ?, ?, ?, ?, ?)");
    foreach (file($log, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) as $line) {
      $e = json_decode($line, true);
      if (!$e || !isset($newId[$e['qid']])) continue;
      $when = (new DateTime($e['t']))->setTimezone(new DateTimeZone('Asia/Kolkata'))->format('Y-m-d H:i:s');
      $insPA->execute([(string)$userId, $newId[$e['qid']], (string)$courseId, ord($e['choice']) - 97, $e['correct'] ? 1 : 0, (int)round($e['seconds']), $when]);
      $imported++;
    }
  }

  echo "course $courseId ($slug), " . count($subjectId) . " subjects, " . count($moduleId) . " topics, " . count($newId) . " questions, user $userId enrolled, $imported practice answers imported\n";
  if ($dry) { $pdo->rollBack(); echo "dry run: rolled back\n"; } else { $pdo->commit(); echo "committed\n"; }
} catch (Throwable $e) {
  $pdo->rollBack();
  echo "rolled back: ", $e->getMessage(), "\n";
  exit(1);
}
