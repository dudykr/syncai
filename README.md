# SyncAI

SyncAI는 Cursor IDE의 커스텀 인스트럭션을 다른 AI 개발 도구용 형식으로 변환해주는 도구입니다.

## 지원하는 AI 도구

- **Cursor IDE** - 원본 형식 유지
- **WindSurf** - 글로벌 룰만 지원
- **Roo Code** - 글로벌 및 폴더별 룰 지원
- **Cline** - 글로벌 룰만 지원

## 설치

```bash
go install github.com/dudykr/syncai@latest
```

또는 소스에서 빌드:

```bash
git clone https://github.com/dudykr/syncai.git
cd syncai
go build -o syncai main.go
```

## 사용법

### Build 명령어

Cursor 룰을 다른 AI 도구 형식으로 변환합니다.

```bash
# 단일 타겟으로 빌드
syncai build --target roo-code

# 여러 타겟으로 빌드
syncai build --target windsurf --target cline

# Watch 모드로 실행 (파일 변경 시 자동 재빌드)
syncai build --target roo-code --watch

# 커스텀 출력 디렉토리 지정
syncai build --target roo-code --output ./output

# 특정 프로젝트 디렉토리 지정
syncai build --target roo-code --project /path/to/project
```

### Import 명령어

다른 프로젝트의 Cursor 룰을 현재 프로젝트로 가져옵니다.

```bash
# 다른 프로젝트에서 룰 가져오기
syncai import /path/to/source/project

# 특정 타겟 디렉토리로 가져오기
syncai import /path/to/source/project --project /path/to/target/project
```

## 프로젝트 구조

SyncAI는 다음과 같은 Cursor IDE 파일들을 인식합니다:

```
project/
├── .cursorrules                    # 글로벌 룰
├── src/
│   └── .cursor/
│       └── rules/
│           ├── rules               # 폴더별 룰
│           ├── frontend.mdc        # MDC 룰 파일
│           └── testing.mdc         # MDC 룰 파일
└── docs/
    └── .cursor/
        └── rules/
            └── documentation.mdc   # MDC 룰 파일
```

## MDC 룰 파일 형식

MDC (Markdown with Configuration) 파일은 YAML frontmatter를 포함할 수 있습니다:

```markdown
---
name: Testing Guidelines
description: Rules for writing and organizing tests
alwaysApply: false
globs:
  - "**/*.test.ts"
  - "**/*.spec.ts"
  - "__tests__/**/*"
---

# Testing Guidelines

Your testing rules content here...
```

### MDC 속성

- `name`: 룰의 이름
- `description`: 룰에 대한 설명
- `alwaysApply`: true이면 항상 적용, false이면 조건부 적용
- `globs`: 이 룰이 적용될 파일 패턴 배열

## 출력 형식

### Cursor IDE
원본 구조와 동일하게 복사됩니다.

### WindSurf
```
dist/windsurf/
└── .windsurfrules
```

### Roo Code
```
dist/roo-code/
├── roo-code-rules.md
├── frontend/
│   └── .roo-rules.md
└── backend/
    └── .roo-rules.md
```

### Cline
```
dist/cline/
└── .cline/
    └── instructions.md
```

## 고급 기능

### Watch 모드

Watch 모드에서는 다음 파일들의 변경사항을 감지합니다:

- `.cursorrules` 파일
- `.cursor/rules/` 디렉토리의 모든 파일
- `*.mdc` 파일

변경 감지 시 500ms의 디바운스 후 자동으로 재빌드됩니다.

### 병렬 처리

여러 타겟을 지정하면 각 타겟별로 병렬로 변환 작업이 수행됩니다.

### 로깅

```bash
# 상세한 로그 출력
syncai build --target roo-code --verbose
```

## 예제

프로젝트에는 두 개의 예제가 포함되어 있습니다:

### Import 테스트
```bash
cd examples/import-test
syncai import ../other-project
```

### Build 테스트
```bash
cd examples/build-test
syncai build --target roo-code --target windsurf
```

## 개발

### 의존성

- Go 1.24+
- github.com/spf13/cobra - CLI 프레임워크
- github.com/fsnotify/fsnotify - 파일 시스템 감시
- github.com/sirupsen/logrus - 로깅
- gopkg.in/yaml.v3 - YAML 파싱

### 테스트 실행

```bash
go test ./...
```

### 빌드

```bash
go build -o syncai main.go
```

## 기여하기

1. 이 저장소를 Fork합니다
2. 기능 브랜치를 만듭니다 (`git checkout -b feature/amazing-feature`)
3. 변경사항을 커밋합니다 (`git commit -m 'Add amazing feature'`)
4. 브랜치에 Push합니다 (`git push origin feature/amazing-feature`)
5. Pull Request를 만듭니다

## 라이센스

이 프로젝트는 MIT 라이센스 하에 배포됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 문제 해결

### 일반적인 문제들

**Q: MDC 파일이 올바르게 파싱되지 않습니다.**
A: YAML frontmatter가 올바른 형식인지 확인하세요. `---`로 시작하고 끝나야 하며, 유효한 YAML 형식이어야 합니다.

**Q: Watch 모드에서 변경사항이 감지되지 않습니다.**
A: 파일이 `.cursorrules`, `.cursor/rules/` 디렉토리 내부, 또는 `.mdc` 확장자를 가지는지 확인하세요.

**Q: 특정 폴더의 룰이 변환되지 않습니다.**
A: 대상 AI 도구가 폴더별 룰을 지원하는지 확인하세요. WindSurf와 Cline은 글로벌 룰만 지원합니다.

### 지원

문제가 발생하면 [GitHub Issues](https://github.com/dudykr/syncai/issues)에서 이슈를 등록해주세요. 