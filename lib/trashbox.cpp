#include <Windows.h>
#include <Shlobj.h>
#include <iostream>
#include <guiddef.h>
#include <fcntl.h>

#include "trashbox.h"

// https://memos-by-oxalis.hatenablog.com/entry/2019/05/01/183947
// https://www.codeproject.com/Articles/2783/How-to-programmatically-use-the-Recycle-Bin
int PrintTrashBox() {
    // Initialize COM
    CoInitialize(NULL);

    // https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/setmode?view=msvc-160
    _setmode(fileno(stdout), _O_U8TEXT);

    // Retrieves the IShellFolder interface for the desktop folder.
    IShellFolder* pDesktopFolder;
    HRESULT hr;
    hr = SHGetDesktopFolder(&pDesktopFolder);
    if (FAILED(hr)) {
        std::cerr << "Failed to get desktop folder." << std::endl;
        CoUninitialize();
        return 1;
    }

    // Get the ITEMIDLIST corresponding to CSIDL_BITBUCKET.
    LPITEMIDLIST pRecycleBinIDL;
    hr = SHGetSpecialFolderLocation(NULL, CSIDL_BITBUCKET, &pRecycleBinIDL);
    if (FAILED(hr)) {
        std::cerr << "Failed to get Recycle Bin folder location." << std::endl;
        pDesktopFolder->Release();
        CoUninitialize();
        return 1;
    }

    IShellFolder* pRecycleBinFolder;
    GUID IID_IShellFolder = { 0x000214e6, 0x0000, 0x0000, {0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46 }};

    // Get the interface pointer specified by riid(IID_IShellFolder) and store it in pRecycleBinFolder
    hr = pDesktopFolder->BindToObject(pRecycleBinIDL, NULL, IID_IShellFolder, (LPVOID*)&pRecycleBinFolder);
    if (FAILED(hr)) {
        std::cerr << "Failed to bind to Recycle Bin folder." << std::endl;
        pDesktopFolder->Release();
        CoTaskMemFree(pRecycleBinIDL);
        CoUninitialize();
        return 1;
    }

    // Enumerate items in the trashbox from pRecycleBinFolder
    IEnumIDList* pEnum;
    hr = pRecycleBinFolder->EnumObjects(NULL, SHCONTF_FOLDERS | SHCONTF_NONFOLDERS, &pEnum);
    if (SUCCEEDED(hr)) {
        LPITEMIDLIST pItemIDL;
        while (pEnum->Next(1, &pItemIDL, NULL) == S_OK) {
            STRRET pName;
            // Retrieve item name and print it
            if (pRecycleBinFolder->GetDisplayNameOf(pItemIDL, SHGDN_NORMAL, &pName) == S_OK) {
                std::wcout << pName.pOleStr << std::endl;
            }
            CoTaskMemFree(pItemIDL);
        }
        pEnum->Release();
    }
    else {
        std::cerr << "Failed to enumerate Recycle Bin items." << std::endl;
    }

    // Release resource
    pRecycleBinFolder->Release();
    pDesktopFolder->Release();
    CoTaskMemFree(pRecycleBinIDL);
    CoUninitialize();

    return 0;
}