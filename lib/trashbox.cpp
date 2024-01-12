#include <Windows.h>
#include <Shlobj.h>
#include <iostream>
#include <guiddef.h>
#include <fcntl.h>
#include <shlwapi.h>

#include "trashbox.h"

// https://devblogs.microsoft.com/oldnewthing/20110830-00/?p=9773
void PrintDisplayName(IShellFolder* psf, PCUITEMID_CHILD pidl, SHGDNF uFlags, PCTSTR pszLabel){
    // Retrieve item name and print it
    STRRET pName;
    HRESULT hr = psf->GetDisplayNameOf(pidl, uFlags, &pName);
    if (SUCCEEDED(hr)) {
        PTSTR pszName;
        hr = StrRetToStrA(&pName, pidl, &pszName);
        if (SUCCEEDED(hr)) {
            wprintf(L"%s\t: %s\n", pszLabel, pszName);
            CoTaskMemFree(pszName);
        }
    }
}

// Get shell folder
HRESULT GetRecycleBinShellFolder(void** pRecycleBinFolder){
    IShellFolder* pDesktopFolder = NULL;
    HRESULT hr = SHGetDesktopFolder(&pDesktopFolder);
    if (FAILED(hr)) {
        wprintf(L"Failed to get desktop folder.\n");
        return hr;
    }

    // Get the ITEMIDLIST corresponding to CSIDL_BITBUCKET.
    LPITEMIDLIST pRecycleBinIDL;
    hr = SHGetSpecialFolderLocation(NULL, CSIDL_BITBUCKET, &pRecycleBinIDL);
    if (FAILED(hr)) {
        wprintf(L"Failed to get Recycle Bin folder location.\n");
        pDesktopFolder->Release();
        return hr;
    }

    // Get the interface pointer specified by riid(IID_IShellFolder) and store it in pRecycleBinFolder
    hr = pDesktopFolder->BindToObject(pRecycleBinIDL, NULL, IID_IShellFolder2, pRecycleBinFolder);
    if (FAILED(hr)) {
        wprintf(L"Failed to bind to Recycle Bin folder.\n");
        pDesktopFolder->Release();
        CoTaskMemFree(pRecycleBinIDL);
        return hr;
    }
    pDesktopFolder->Release();
    CoTaskMemFree(pRecycleBinIDL);

    return hr;
}


int RestoreItem(char *file) {
    // Initialize COM
    CoInitialize(NULL);

    // https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/setmode?view=msvc-160
    _setmode(fileno(stdout), _O_U8TEXT);
    setlocale(LC_ALL, "Japanese");

    // Retrieves the IShellFolder interface for the desktop folder.
    // https://learn.microsoft.com/ja-jp/windows/win32/api/shobjidl_core/nn-shobjidl_core-ishellfolder2
    IShellFolder2* pRecycleBinFolder;
    HRESULT hr;
    hr = GetRecycleBinShellFolder((void**)&pRecycleBinFolder);
    if (FAILED(hr)) {
        std::cerr << "Failed to GetRecycleBinShellFolder()" << std::endl;
        CoUninitialize();
    }
    
    // Enumerate items in the trashbox from pRecycleBinFolder
    LPITEMIDLIST pItemIDL = nullptr;
    IEnumIDList* pEnum = nullptr;
    hr = pRecycleBinFolder->EnumObjects(nullptr, SHCONTF_NONFOLDERS, &pEnum);
    if (FAILED(hr)) {
        std::cerr << "Failed to pRecycleBinFolder->EnumObjects()" << std::endl;
        CoUninitialize();
    }

    IContextMenu* pMenu = nullptr;
    while (pEnum->Next(1, &pItemIDL, NULL) == S_OK) {
        STRRET pName;
        hr = pRecycleBinFolder->GetDisplayNameOf(pItemIDL, SHGDN_INFOLDER, &pName);
        if (FAILED(hr)) {
            std::cerr << "Failed to GetDisplayNameOf()" << std::endl;
            continue;
        }
        PTSTR pszName;
        hr = StrRetToStrA(&pName, pItemIDL, &pszName);
        if (FAILED(hr)) {
            // Failed to StrRetToStrA
            std::cerr << "Failed to StrRetToStrA()" << std::endl;
            continue;
        }

        // Support for wildcard
        if (PathMatchSpec(pszName, file)){
            hr = pRecycleBinFolder->GetUIObjectOf(nullptr, 1, (LPCITEMIDLIST*)&pItemIDL, IID_IContextMenu, nullptr, (void**)&pMenu);
            PrintDisplayName(pRecycleBinFolder, pItemIDL, SHGDN_NORMAL, TEXT("Restore\t"));

            // http://hiroshi0945.blog75.fc2.com/blog-entry-66.html#_InvokeCommandInRecycleBin%E9%96%A2%E6%95%B0
            CMINVOKECOMMANDINFO ici = { sizeof(CMINVOKECOMMANDINFO) };
            ici.lpVerb = "undelete";
            ici.nShow = SW_NORMAL;
            hr = pMenu->InvokeCommand((CMINVOKECOMMANDINFO*)&ici);
        }
        ILFree(pItemIDL);
        CoTaskMemFree(pszName);
    }

    CoUninitialize();

    return 0;
}