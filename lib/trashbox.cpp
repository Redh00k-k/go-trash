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

// https://devblogs.microsoft.com/oldnewthing/20110830-00/?p=9773
void PrintDetail(IShellFolder2* psf, PCUITEMID_CHILD pidl, const SHCOLUMNID* pscid, PCTSTR pszLabel){
    VARIANT vt;
    HRESULT hr = psf->GetDetailsEx(pidl, pscid, &vt);
    if (SUCCEEDED(hr)) {
        hr = VariantChangeType(&vt, &vt, 0, VT_BSTR);
        if (SUCCEEDED(hr)) {
            wprintf(L"%s\t: %ls\n", pszLabel, V_BSTR(&vt));
        }
        VariantClear(&vt);
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

// https://memos-by-oxalis.hatenablog.com/entry/2019/05/01/183947
// https://www.codeproject.com/Articles/2783/How-to-programmatically-use-the-Recycle-Bin
int PrintTrashBox() {
    // Initialize COM
    CoInitialize(NULL);

    // https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/setmode?view=msvc-160
    _setmode(fileno(stdout), _O_U8TEXT);
    setlocale(LC_ALL, "Japanese");

    // Retrieves the IShellFolder interface for the desktop folder.
    HRESULT hr;
    IShellFolder2* pRecycleBinFolder;
    hr = GetRecycleBinShellFolder((void**)&pRecycleBinFolder);
    if (FAILED(hr)) {
        wprintf(L"Failed to GetRecycleBinShellFolder().\n");
        CoUninitialize();
    }

    // PKEY_Size from propkey.h
    SHCOLUMNID PKEY_Size { { 0xB725F130, 0x47EF, 0x101A, 0xA5, 0xF1, 0x02, 0x60, 0x8C, 0x9E, 0xEB, 0xAC }, 12 };

    // Enumerate items in the trashbox from pRecycleBinFolder
    IEnumIDList* pEnum;
    hr = pRecycleBinFolder->EnumObjects(NULL, SHCONTF_FOLDERS | SHCONTF_NONFOLDERS, &pEnum);
    if (SUCCEEDED(hr)) {
        LPITEMIDLIST pItemIDL;
        while (pEnum->Next(1, &pItemIDL, NULL) == S_OK) {
            const SHCOLUMNID SCID_OriginalLocation = { PSGUID_DISPLACED, PID_DISPLACED_FROM };
            const SHCOLUMNID SCID_DateDeleted = { PSGUID_DISPLACED, PID_DISPLACED_DATE };

            wprintf(L"\n");
            PrintDisplayName(pRecycleBinFolder, pItemIDL, SHGDN_INFOLDER, TEXT("InFolder"));
            PrintDisplayName(pRecycleBinFolder, pItemIDL, SHGDN_NORMAL, TEXT("Normal\t"));
            PrintDisplayName(pRecycleBinFolder, pItemIDL, SHGDN_FORPARSING, TEXT("ForParsing"));
            // PrintDetail(pRecycleBinFolder, pItemIDL, &SCID_OriginalLocation, TEXT("OriginalLocation"));
            PrintDetail(pRecycleBinFolder, pItemIDL, &SCID_DateDeleted, TEXT("DateDeleted"));
            PrintDetail(pRecycleBinFolder, pItemIDL, &PKEY_Size, TEXT("Size\t"));

            CoTaskMemFree(pItemIDL);
        }
        pEnum->Release();
    }
    else {
        wprintf(L"Failed to enumerate Recycle Bin items.\n");
    }

    // Release resource
    pRecycleBinFolder->Release();
    CoUninitialize();

    return 0;
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