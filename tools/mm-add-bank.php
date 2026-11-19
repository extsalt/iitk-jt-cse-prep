<?php
// Adds one runner bank (runner/banks/<id>.json) to the existing IIT Kanpur course in the local MindMantra database.
// It creates missing topic modules, the questions, and an unpublished test course <course slug>-mock-NN holding them.
// Usage: php tools/mm-add-bank.php <bank-id> <mock-number> [course-slug, default iitk-jt-cse-2026] [--dry-run]
foreach (file('D:/mindmantra.live/.env.local', FILE_IGNORE_NEW_LINES) as $l) {
  if (str_contains($l, '=') && !str_starts_with(trim($l), '#')) { [$k, $v] = explode('=', $l, 2); $env[trim($k)] = trim($v); }
}
$pdo = new PDO("mysql:host={$env['MYSQL_HOST']};port={$env['MYSQL_PORT']};dbname={$env['MYSQL_DATABASE']};charset=utf8mb4", $env['MYSQL_USER'], $env['MYSQL_PASSWORD'], [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]);
$args = array_values(array_filter($argv, fn($a) => !str_starts_with($a, '--')));
$bankId = $args[1] ?? '';
$num = $args[2] ?? '';
$dry = in_array('--dry-run', $argv, true);
$file = "D:/iit-kanpur-lab/runner/banks/$bankId.json";
if (!is_file($file) || !preg_match('/^\d\d$/', $num)) exit("usage: php mm-add-bank.php <bank-id> <two-digit mock number> [--dry-run]\n");
$bank = json_decode(file_get_contents($file), true);

$slug = $args[3] ?? 'iitk-jt-cse-2026';
$course = $pdo->query("SELECT id FROM courses WHERE slug = " . $pdo->quote($slug))->fetch();
if (!$course) exit("course $slug not found\n");
$courseId = (int)$course['id'];
$testSlug = "$slug-mock-$num";
if ($pdo->query("SELECT id FROM courses WHERE slug = " . $pdo->quote($testSlug))->fetch()) exit("$testSlug already exists; nothing done\n");

preg_match_all('/\{"([^"]+)", \[\]string\{([^}]*)\}\}/s', file_get_contents('D:/iit-kanpur-lab/runner/topics.go'), $m, PREG_SET_ORDER);
$areaOf = [];
foreach ($m as $row) {
  preg_match_all('/"([^"]+)"/', $row[2], $tm);
  foreach ($tm[1] as $j => $t) $areaOf[$t] = [$row[1], $j];
}
$clean = fn(string $s) => str_replace(['**', '`'], '', $s);
$slugify = fn(string $s) => trim(preg_replace('/[^a-z0-9]+/', '-', strtolower($s)), '-');

$pdo->beginTransaction();
try {
  $subjectId = []; $moduleId = [];
  foreach ($bank['questions'] as $q) {
    $topic = $q['topic'];
    if (!isset($areaOf[$topic])) throw new Exception("topic not in topics.go: $topic");
    if ($pdo->query("SELECT id FROM questions WHERE question_id = " . $pdo->quote($q['id']))->fetch()) throw new Exception("question {$q['id']} already imported");
    [$area, $ord] = $areaOf[$topic];
    if (!isset($subjectId[$area])) {
      $s = $pdo->query("SELECT id FROM subjects WHERE subject_code = " . $pdo->quote('iitk-' . $slugify($area)))->fetch();
      if (!$s) throw new Exception("area subject missing: $area");
      $subjectId[$area] = (int)$s['id'];
    }
    if (!isset($moduleId[$topic])) {
      $mm = $pdo->query("SELECT id FROM subject_modules WHERE module_code = " . $pdo->quote('iitk-' . $slugify($topic)))->fetch();
      if ($mm) {
        $moduleId[$topic] = (int)$mm['id'];
      } else {
        $pdo->prepare("INSERT INTO subject_modules (module_title, module_code, subject_id, order_index, status) VALUES (?, ?, ?, ?, 1)")->execute([$topic, 'iitk-' . $slugify($topic), $subjectId[$area], $ord]);
        $moduleId[$topic] = (int)$pdo->lastInsertId();
        echo "new topic module: $topic\n";
      }
    }
  }

  $minutes = (int)$bank['minutes'];
  $pdo->prepare("INSERT INTO courses (slug, title, description, time_limit, total_marks, total_questions, negative_marking, is_published, category) VALUES (?, ?, ?, ?, ?, ?, 0.25, 0, 'mock_test')")
    ->execute([$testSlug, ($slug === 'iitk-jt-cse-2026' ? 'IITK JT (CSE) ' : '') . $bank['title'], $bank['title'] . ': ' . count($bank['questions']) . ' questions in ' . $minutes . ' minutes.', $minutes, count($bank['questions']), count($bank['questions'])]);
  $testId = (int)$pdo->lastInsertId();

  $insQ = $pdo->prepare("INSERT INTO questions (question_text, options, correct_answer, subject, subject_id, subject_module_id, year, quality_status, reviewed_by, reviewed_at, is_active, question_id, explanation, ai_review_status) VALUES (?, ?, ?, ?, ?, ?, 2026, 'verified', 'claude-verified', NOW(), 1, ?, ?, 'none')");
  $insCQ = $pdo->prepare("INSERT INTO course_questions (course_id, question_id, is_active) VALUES (?, ?, 1)");
  foreach ($bank['questions'] as $q) {
    $text = (!empty($q['passage']) ? "Passage: " . $clean($q['passage']) . "\n\n" : '') . $clean($q['question']) . (!empty($q['code']) ? "\n\n" . $q['code'] : '');
    $exp = $clean($q['explanation']) . (!empty($q['source']) ? ' [Source: ' . $q['source'] . ']' : '');
    $insQ->execute([$text, json_encode(array_map($clean, $q['options']), JSON_UNESCAPED_UNICODE), ord($q['answer']) - 97, $q['topic'], $subjectId[$areaOf[$q['topic']][0]], $moduleId[$q['topic']], $q['id'], $exp]);
    $insCQ->execute([$testId, (int)$pdo->lastInsertId()]);
  }
  echo "$testSlug: test course $testId, " . count($bank['questions']) . " questions\n";
  if ($dry) { $pdo->rollBack(); echo "dry run: rolled back\n"; } else { $pdo->commit(); echo "committed\n"; }
} catch (Throwable $e) {
  $pdo->rollBack();
  echo "rolled back: ", $e->getMessage(), "\n";
  exit(1);
}
