#include "FileDialog.h"

#ifdef _WIN32
#include <windows.h>
#include <commdlg.h>

bool OpenFileDialog(char* outPath, int maxLen) {
    OPENFILENAME ofn;
    ZeroMemory(&ofn, sizeof(ofn));
    ofn.lStructSize = sizeof(ofn);
    ofn.hwndOwner = NULL;
    ofn.lpstrFile = outPath;
    ofn.nMaxFile = maxLen;
    ofn.lpstrFilter = "Mesh Files\0*.msh;*.txt\0All Files\0*.*\0";
    ofn.nFilterIndex = 1;
    ofn.lpstrInitialDir = NULL;
    ofn.Flags = OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST;
    return GetOpenFileName(&ofn);
}
#endif
