import SideNav from './SideNav'
import NavBar from './NavBar'

function Settings(): JSX.Element {
  return (
    <div className="flex ml-52">
      <SideNav />
      {/* <NavBar /> */}
      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Settings</h1>
        <div className="border-b border-gray-300 mb-6"></div>

        <h2 className="text-xl font-semibold mb-2">Transfer</h2>
        <div className="border-b border-gray-300 mb-4 w-1/6"></div>

        <div className="mb-4">
          <label htmlFor="transfer-name" className="block text-sm font-medium text-gray-700 mb-1">
            Save Folder
          </label>
          <input
            type="text"
            id="transfer-name"
            className="mt-1 block w-1/3 border border-gray-300 rounded-md p-2"
            placeholder="Enter new save directory"
          />
        </div>

        <div className="mb-4">
          <label htmlFor="upload-limit" className="block text-sm font-medium text-gray-700 mb-1">
            Upload Limit
          </label>
          <div className="flex border border-gray-300 rounded-md p-2 w-1/3 items-center">
            <input
              type="number"
              id="upload-limit"
              className="block border-none outline-none w-full pr-4"
              placeholder="0"
              min="0"
              step="any"
            />
            <span className="text-gray-500">Mbps</span>
          </div>
        </div>

        <div className="mb-4">
          <label htmlFor="upload-limit" className="block text-sm font-medium text-gray-700 mb-1">
            Upload Limit
          </label>
          <div className="flex border border-gray-300 rounded-md p-2 w-1/3 items-center">
            <input
              type="number"
              id="upload-limit"
              className="block border-none outline-none w-full pr-4"
              placeholder="0"
              min="0"
              step="any"
            />
            <span className="text-gray-500">Mbps</span>
          </div>
        </div>

        <button className="mt-4 px-4 py-2 bg-[#737fa3] text-white font-semibold rounded-md hover:bg-[#7c85a3]">
          Save Changes
        </button>
      </div>
    </div>
  )
}

export default Settings
