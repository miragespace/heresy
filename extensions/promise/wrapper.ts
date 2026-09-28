const __runtimeResolverResult = async <T>(
  arg: T | Promise<T>, // argument as native object
  resolve: (result: T) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  try {
    const r = await arg;
    resolve(r);
  } catch (e) {
    reject(e);
  }
};

const __runtimeResolverFuncWithArg = async <T>(
  fn: (arg: T) => T | Promise<T>, // JavaScript native function, usually the handler in the script
  arg: T, // argument to the said handler as native object
  resolve: (result: T) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  try {
    resolve(await fn(arg));
  } catch (e) {
    reject(e);
  }
};
