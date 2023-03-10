function normalizeName(name: string) {
  if (typeof name !== "string") {
    name = String(name);
  }

  name = name.trim();

  if (name.length === 0) {
    throw new TypeError("Header field name is empty");
  }

  if (/[^a-z0-9\-#$%&'*+.^_`|~!]/i.test(name)) {
    throw new TypeError(`Invalid character in header field name: ${name}`);
  }

  return name.toLowerCase();
}

function normalizeValue(value: string | number) {
  if (typeof value !== "string") {
    value = String(value);
  }
  return value;
}

export type HeadersInit =
  | Headers
  | Record<string, string>
  | [key: string, value: string][];

class Headers {
  map: Map<string, string> = new Map();

  constructor(init: unknown) {
    if (init instanceof Headers) {
      init.forEach((value: string, name: string) => {
        this.append(name, value);
      });

      return this;
    }

    if (Array.isArray(init)) {
      init.forEach(([name, value]) => {
        this.append(name, value);
      });

      return this;
    }

    Object.getOwnPropertyNames(init).forEach((name) =>
      this.append(name, (init as Record<string, string>)[name])
    );
  }

  append(name: string, value: string): void {
    name = normalizeName(name);
    value = normalizeValue(value);
    const oldValue = this.get(name);
    // From MDN: If the specified header already exists and accepts multiple values, append() will append the new value to the end of the value set.
    // However, we're a missing a check on whether the header does indeed accept multiple values
    this.map.set(name, oldValue ? oldValue + ", " + value : value);
  }

  delete(name: string): void {
    this.map.delete(normalizeName(name));
  }

  get(name: string): string | null {
    name = normalizeName(name);
    return this.map.get(name) ?? null;
  }

  has(name: string): boolean {
    return this.map.has(normalizeName(name));
  }

  set(name: string, value: string | number): void {
    this.map.set(normalizeName(name), normalizeValue(value));
  }

  forEach(
    callback: (value: string, key: string, parent?: object) => void,
    thisArg?: object
  ): void {
    this.map.forEach((value, name) => {
      callback.call(thisArg, value, name, this);
    }, this);
  }

  keys(): IterableIterator<string> {
    return this.map.keys();
  }

  values(): IterableIterator<string> {
    return this.map.values();
  }

  entries(): IterableIterator<[string, string]> {
    return this.map.entries();
  }

  [Symbol.iterator](): IterableIterator<[string, string]> {
    return this.entries();
  }
}

export default Headers;
