const __runtimeResolver = (
  fn: (arg: any) => any | Promise<any>,
  arg: any,
  resolve: (result: any) => void,
  reject: (e: any) => void
) => {
  Promise.resolve(fn(arg)).then(resolve).catch(reject);
};
