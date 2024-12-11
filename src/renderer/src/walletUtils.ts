export const tranferMoney = async (address: string, amount: number) => {
  const r = await fetch('http://localhost:8080/transfer', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ account: 'default', address, amount })
  })

  const data = await r.json()

  if (data.status != 'success') return 'No money'

  return data.message
}
