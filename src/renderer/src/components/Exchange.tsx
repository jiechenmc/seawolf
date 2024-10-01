import SideNav from './SideNav'

function Exchange(): JSX.Element {
  return (
    <div className="flex ml-52">
      <SideNav />

      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Getting Files</h1>

        <div className="flex justify-between mb-16 w-1/3">
          <div className="bg-white p-4 rounded-lg shadow-md w-full">
            <h2 className="text-xl font-semibold">Download File</h2>

            <div className="mt-7">
              <label className="block text-sm font-medium text-gray-700 mb-2">File Hash ID</label>
              <input
                type="text"
                className="mt-1 block w-1/3 border border-gray-300 rounded-md p-2"
                placeholder="Hash ID"
              />
            </div>
            <div className="mt-5">
              <label className="block text-sm font-medium text-gray-700 mb-2">Amount</label>
              <input
                type="number"
                className="mt-1 block w-1/3 border border-gray-300 rounded-md p-2"
                placeholder="0"
                min="0"
              />
            </div>
            <button className="mt-7 px-4 py-2 bg-[#737fa3] text-white font-semibold rounded-md hover:bg-[#7c85a3]">
              Get File
            </button>
          </div>
        </div>
        <h2 className="text-xl font-semibold mb-2">Processes</h2>
        <div className="overflow-x-auto border border-gray-300 rounded-lg">
          <div className="flex items-center p-2 border-b border-gray-300">
            <span className="flex-1 font-semibold">File</span>
            <span className="flex-1 font-semibold">Bytes</span>
            <span className="flex-1 font-semibold">ETA</span>
            <span className="flex-1 font-semibold">Status</span>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Exchange
