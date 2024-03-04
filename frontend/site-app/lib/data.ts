export async function fetchData(id: string | undefined, include: string | undefined): Promise<any> {

  const data = {
    id: id || '',
    include: include || ''
  };

  return new Promise(resolve => {
    setTimeout(() => {
      resolve(data);
    }, 1000)
  });
}
