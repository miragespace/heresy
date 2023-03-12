const __runtimeResolverFuncWithArg = <T>(
  fn: (arg: T) => T | Promise<T>, // JavaScript native function, usually the handler in the script
  arg: any, // argument to the said handler as native object
  resolve: (result: T) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  Promise.resolve(fn(arg)).then(resolve).catch(reject);
};

const __runtimeResolverFuncWithSpread = <T>(
  fn: (arg: T) => T[] | Promise<T[]>, // JavaScript native function, usually the handler in the script
  arg: any, // argument to the said handler as native object
  resolve: (...result: T[]) => void, // callback to Go when the Promise resolves
  reject: (e: unknown) => void // callback to Go when the Promise rejects
) => {
  Promise.resolve(fn(arg))
    .then((r) => {
      resolve(...r);
    })
    .catch(reject);
};
