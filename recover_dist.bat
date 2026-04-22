cd /d D:\gotest\s2qt
if not exist frontend\dist mkdir frontend\dist
(
  echo ^<!doctype html^>
  echo ^<html^>
  echo ^<head^>^<meta charset="utf-8" /^>^<title^>placeholder^</title^>^</head^>
  echo ^<body^>placeholder^</body^>
  echo ^</html^>
) > frontend\dist\index.html