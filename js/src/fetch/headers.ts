function normalizeName(name: string | number) {
  if (typeof name !== "string") {
    name = String(name);
  }
  if (/[^a-z0-9\-#$%&'*+.^_`|~!]/i.test(name) || name === "") {
    throw new TypeError(
      'Invalid character in header field name: "' + name + '"'
    );
  }
  return name.toLowerCase();
}

function normalizeValue(value: string | number) {
  if (typeof value !== "string") {
    value = String(value);
  }
  return value;
}

function iteratorFor<T, R>(items: T[]) {
  var iterator: IterableIterator<R> = {
    // @ts-ignore
    next() {
      var value = items.shift();
      return { done: value === undefined, value: value };
    },
    [Symbol.iterator]() {
      return iterator;
    },
  };

  return iterator;
}

export default class Headers {
  map: Record<string, string> = {};

  constructor(headers: unknown) {
    if (headers instanceof Headers) {
      headers.forEach((value, name) => {
        this.append(name, value);
      });
    } else if (Array.isArray(headers)) {
      headers.forEach((header) => {
        this.append(header[0], header[1]);
      });
    } else if (headers) {
      Object.getOwnPropertyNames(headers).forEach((name: string) => {
        this.append(name, (headers as any)[name]);
      });
    }
  }

  append(name: string | number, value: string | number) {
    name = normalizeName(name);
    value = normalizeValue(value);
    var oldValue = this.map[name];
    this.map[name] = oldValue ? oldValue + ", " + value : value;
  }

  delete(name: string | number) {
    delete this.map[normalizeName(name)];
  }

  has(name: string | number): boolean {
    return this.map.hasOwnProperty(normalizeName(name));
  }

  get(name: string | number): string | null {
    name = normalizeName(name);
    return this.has(name) ? this.map[name] : null;
  }

  set(name: string | number, value: string | number) {
    this.map[normalizeName(name)] = normalizeValue(value);
  }

  forEach(
    callback: (value: string, key: string, parent?: object) => void,
    thisArg?: object
  ) {
    for (var name in this.map) {
      if (this.map.hasOwnProperty(name)) {
        callback.call(thisArg, this.map[name], name, this);
      }
    }
  }

  keys(): IterableIterator<string> {
    var items: string[] = [];
    this.forEach(function (value, name) {
      items.push(name);
    });
    return iteratorFor(items);
  }

  values(): IterableIterator<string> {
    var items: string[] = [];
    this.forEach(function (value) {
      items.push(value);
    });
    return iteratorFor(items);
  }

  entries(): IterableIterator<[string, string]> {
    var items: string[][] = [];
    this.forEach(function (value, name) {
      items.push([name, value]);
    });
    return iteratorFor(items);
  }

  [Symbol.iterator]() {
    return this.entries();
  }
}
