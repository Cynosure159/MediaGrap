import { onMounted, onUnmounted, ref, watch } from "vue";
import { settings } from "@/api/library";

export const movieRenameDefault = "${title} (${year})/${title} (${year})";
export const tvRenameDefault =
  "${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}";
export const movieRenameTokens = [
  "title",
  "originalTitle",
  "year",
  "resolution",
  "videoCodec",
  "audioCodec",
  "edition",
  "imdbId",
] as const;
export const tvRenameTokens = [
  "showTitle",
  "showOriginalTitle",
  "originalTitle",
  "seasonNumber",
  "seasonNumberPad",
  "episodeNumber",
  "episodeNumberPad",
  "episodeTitle",
  "year",
  "resolution",
  "videoCodec",
  "audioCodec",
] as const;

export interface RenameExpression {
  token: string;
  prefix: string;
  suffix: string;
  optional: boolean;
}

export interface RenamePart {
  literal?: string;
  expression?: RenameExpression;
}

export type RenamePatternErrorCode =
  | "unsafe_pattern"
  | "malformed_expression"
  | "nested_expression"
  | "comma_count"
  | "unsafe_affix"
  | "malformed_token"
  | "unsupported_token"
  | "missing_value"
  | "invalid_path_segment"
  | "empty_filename"
  | "invalid_directory_segment";

export class RenamePatternError extends Error {
  constructor(
    message: string,
    readonly code: RenamePatternErrorCode,
    readonly token?: string,
  ) {
    super(message);
    this.name = "RenamePatternError";
  }
}

function isControlCodePoint(codePoint: number): boolean {
  return (
    (codePoint >= 0 && codePoint <= 0x1f) ||
    (codePoint >= 0x7f && codePoint <= 0x9f)
  );
}

// Match Go's unicode.IsSpace (unicode.White_Space plus the Latin-1 controls),
// deliberately excluding JavaScript trim's extra U+FEFF behavior.
function isGoSpace(codePoint: number): boolean {
  return (
    (codePoint >= 0x09 && codePoint <= 0x0d) ||
    codePoint === 0x20 ||
    codePoint === 0x85 ||
    codePoint === 0xa0 ||
    codePoint === 0x1680 ||
    (codePoint >= 0x2000 && codePoint <= 0x200a) ||
    codePoint === 0x2028 ||
    codePoint === 0x2029 ||
    codePoint === 0x202f ||
    codePoint === 0x205f ||
    codePoint === 0x3000
  );
}

function trimGoSpace(value: string): string {
  const chars = [...value];
  let start = 0;
  let end = chars.length;
  while (start < end && isGoSpace(chars[start]!.codePointAt(0)!)) start++;
  while (end > start && isGoSpace(chars[end - 1]!.codePointAt(0)!)) end--;
  return chars.slice(start, end).join("");
}

function utf8ByteLength(value: string): number {
  let length = 0;
  for (const char of value) {
    const codePoint = char.codePointAt(0)!;
    length +=
      codePoint <= 0x7f
        ? 1
        : codePoint <= 0x7ff
          ? 2
          : codePoint <= 0xffff
            ? 3
            : 4;
  }
  return length;
}

function collapseGoSpace(value: string): string {
  let result = "";
  let pendingSpace = false;
  for (const char of value) {
    if (isGoSpace(char.codePointAt(0)!)) {
      pendingSpace = true;
      continue;
    }
    if (pendingSpace && result) result += " ";
    pendingSpace = false;
    result += char;
  }
  return result;
}

function containsControl(value: string): boolean {
  return [...value].some((char) => isControlCodePoint(char.codePointAt(0)!));
}

function hasUnsafePatternChars(value: string): boolean {
  return containsControl(value) || /[\\:*?"<>|]/.test(value);
}

function hasUnsafeAffixChars(value: string): boolean {
  return containsControl(value) || /[\\/:*?"<>|{}]/.test(value);
}

const tokenName = /^[A-Za-z][A-Za-z0-9]*$/;

function patternError(
  code: RenamePatternErrorCode,
  message: string,
  token?: string,
): RenamePatternError {
  return new RenamePatternError(message, code, token);
}

export function parseRenamePattern(
  pattern: string,
  allowed: readonly string[],
): RenamePart[] {
  if (
    !pattern ||
    utf8ByteLength(pattern) > 1024 ||
    hasUnsafePatternChars(pattern) ||
    pattern.startsWith("/")
  ) {
    throw patternError(
      "unsafe_pattern",
      "rename pattern must be a safe relative path",
    );
  }
  const accepted = new Set(allowed);
  const parts: RenamePart[] = [];
  let literalStart = 0;
  let cursor = 0;
  while (cursor < pattern.length) {
    const start = pattern.indexOf("${", cursor);
    if (start < 0) break;
    if (start > literalStart)
      parts.push({ literal: pattern.slice(literalStart, start) });
    const end = pattern.indexOf("}", start + 2);
    if (end < 0)
      throw patternError("malformed_expression", "malformed naming expression");
    const body = pattern.slice(start + 2, end);
    if (body.includes("${") || body.includes("}"))
      throw patternError(
        "nested_expression",
        "nested naming expressions are not supported",
      );
    const commas = [...body].filter((char) => char === ",").length;
    let expression: RenameExpression;
    if (commas === 0) {
      expression = {
        token: trimGoSpace(body),
        prefix: "",
        suffix: "",
        optional: false,
      };
    } else if (commas === 2) {
      const [prefix, token, suffix] = body.split(",");
      if (
        hasUnsafeAffixChars(prefix) ||
        hasUnsafeAffixChars(suffix) ||
        prefix === "." ||
        prefix === ".." ||
        suffix === "." ||
        suffix === ".."
      ) {
        throw patternError(
          "unsafe_affix",
          "optional naming affixes must be safe filename text",
        );
      }
      expression = {
        token: trimGoSpace(token),
        prefix,
        suffix,
        optional: true,
      };
    } else {
      throw patternError(
        "comma_count",
        "naming expressions must contain either no commas or exactly two commas",
      );
    }
    if (!tokenName.test(expression.token))
      throw patternError(
        "malformed_token",
        `malformed naming token "${expression.token}"`,
        expression.token,
      );
    if (!accepted.has(expression.token))
      throw patternError(
        "unsupported_token",
        `unsupported naming token "${expression.token}"`,
        expression.token,
      );
    parts.push({ expression });
    cursor = end + 1;
    literalStart = cursor;
  }
  if (literalStart < pattern.length)
    parts.push({ literal: pattern.slice(literalStart) });
  if (
    parts.some(
      (part) => part.literal !== undefined && /[${}]/.test(part.literal),
    )
  )
    throw patternError("malformed_expression", "malformed naming expression");
  for (const segment of pattern.split("/")) {
    if (!segment || segment === "." || segment === "..")
      throw patternError(
        "invalid_path_segment",
        "rename pattern contains an invalid path segment",
      );
  }
  return parts;
}

export function renderRenamePattern(
  pattern: string,
  values: Record<string, string>,
  allowed: readonly string[],
): string {
  const parts = parseRenamePattern(pattern, allowed);
  return parts
    .map((part) => {
      if (part.literal !== undefined) return part.literal;
      const expression = part.expression!;
      const rawValue = values[expression.token] ?? "";
      if (!trimGoSpace(rawValue)) {
        if (expression.optional) return "";
        throw patternError(
          "missing_value",
          `naming token "${expression.token}" has no available value`,
          expression.token,
        );
      }
      const value = rawValue.replace(/[\\/]/g, " ");
      return expression.optional
        ? `${expression.prefix}${value}${expression.suffix}`
        : value;
    })
    .join("");
}

function sanitizeFilename(value: string): string {
  return collapseGoSpace(
    value.replace(/[<>:"/\\|?*\u0000-\u001f]/g, " "),
  ).replace(/^[. ]+|[. ]+$/g, "");
}

export interface RenamePatternSimulation {
  dir: string;
  filename: string;
  error?: string;
  errorCode?: RenamePatternErrorCode;
  errorToken?: string;
}

export function simulateRenamePattern(
  pattern: string,
  values: Record<string, string>,
  allowed: readonly string[],
): RenamePatternSimulation {
  try {
    const rendered = renderRenamePattern(pattern, values, allowed);
    const segments = rendered.split("/").map(sanitizeFilename);
    if (!segments.at(-1))
      throw patternError(
        "empty_filename",
        "naming pattern produced an empty filename",
      );
    if (
      segments
        .slice(0, -1)
        .some((segment) => !segment || segment === "." || segment === "..")
    ) {
      throw patternError(
        "invalid_directory_segment",
        "naming pattern produced an invalid directory segment",
      );
    }
    const filename = `${segments.at(-1)!}.mkv`;
    return {
      dir: segments.slice(0, -1).join("/") + (segments.length > 1 ? "/" : ""),
      filename,
    };
  } catch (error) {
    return {
      dir: "",
      filename: "—",
      error: error instanceof Error ? error.message : "invalid naming pattern",
      errorCode: error instanceof RenamePatternError ? error.code : undefined,
      errorToken: error instanceof RenamePatternError ? error.token : undefined,
    };
  }
}

export function renamePatternErrorMessage(
  code: RenamePatternErrorCode | undefined,
  token: string | undefined,
  zh: boolean,
  fallback: string,
): string {
  if (!code) return fallback;
  const messages: Record<RenamePatternErrorCode, [string, string]> = {
    unsafe_pattern: [
      "Rename pattern contains unsafe characters",
      "重命名规则包含不安全字符",
    ],
    malformed_expression: ["Malformed naming expression", "命名表达式格式错误"],
    nested_expression: [
      "Nested naming expressions are not supported",
      "不支持嵌套命名表达式",
    ],
    comma_count: [
      "Expressions must contain either no commas or exactly two commas",
      "表达式必须不含逗号或恰好包含两个逗号",
    ],
    unsafe_affix: [
      "Optional affixes contain unsafe filename text",
      "可选表达式的前后缀包含不安全文件名字符",
    ],
    malformed_token: ["Malformed naming token", "命名占位符格式错误"],
    unsupported_token: ["Unsupported naming token", "不支持的命名占位符"],
    missing_value: [
      "A required token has no available value",
      "必需占位符没有可用值",
    ],
    invalid_path_segment: [
      "Rename pattern contains an invalid path segment",
      "重命名规则包含无效路径段",
    ],
    empty_filename: [
      "The pattern produced an empty filename",
      "规则生成了空文件名",
    ],
    invalid_directory_segment: [
      "The pattern produced an invalid directory segment",
      "规则生成了无效目录段",
    ],
  };
  const message = messages[code][zh ? 1 : 0];
  return token && (code === "unsupported_token" || code === "missing_value")
    ? `${message}: ${token}`
    : message;
}

export function useRenamePattern(kind: "movie" | "tv") {
  const pattern = ref(kind === "movie" ? movieRenameDefault : tvRenameDefault);
  const selectedPreset = ref("custom");
  let edited = false;
  let disposed = false;
  watch(
    pattern,
    () => {
      edited = true;
    },
    { flush: "sync" },
  );
  onUnmounted(() => {
    disposed = true;
  });
  onMounted(async () => {
    try {
      const current = await settings();
      const saved =
        kind === "movie" ? current.movieRenamePattern : current.tvRenamePattern;
      if (!disposed && !edited && saved) pattern.value = saved;
    } catch {
      // Keep the built-in preview template when settings cannot be loaded.
    }
  });
  return { pattern, selectedPreset };
}
